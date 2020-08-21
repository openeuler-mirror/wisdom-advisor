/*
 * Copyright (c) 2020 Huawei Technologies Co., Ltd.
 * wisdom-advisor is licensed under the Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *     http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
 * PURPOSE.
 * See the Mulan PSL v2 for more details.
 * Create: 2020-6-9
 */

// Package policy provides methods to set affinity of tasks
package policy

import (
	"container/list"
	"gitee.com/wisdom-advisor/common/netaffinity"
	"gitee.com/wisdom-advisor/common/procscan"
	"gitee.com/wisdom-advisor/common/sched"
	"gitee.com/wisdom-advisor/common/sysload"
	"gitee.com/wisdom-advisor/common/threadaffinity"
	"gitee.com/wisdom-advisor/common/topology"
	"gitee.com/wisdom-advisor/common/utils"
	log "github.com/sirupsen/logrus"
	"sync/atomic"
)

type bindGroupInfo struct {
	numaNode *topology.TopoNode
	bindNode *topology.TopoNode
	name     string
	tasks    list.List
}

// ControlBlock  is to control the behaviour of policy
type ControlBlock struct {
	ptraceScanSwitch int32
}

type bindTaskInfo struct {
	group       *bindGroupInfo
	groupEle    *list.Element
	nodeEle     *list.Element
	retryEle    *list.Element
	oldBindNode *topology.TopoNode
	newBindNode *topology.TopoNode
	tid         uint64
}

var bindTaskMap = make(map[uint64]*bindTaskInfo)
var bindGroupMap = make(map[string]*bindGroupInfo)
var retryTasks list.List

var netAwareOn = false
var numaAwareOn = false
var cclAwareOn = false
var coarseGrain = false

const (
	cpuGapPct       = 30
	toPercentage    = 100
	eventsBufferLen = 5000
)

var affinityAware = false
var taskName string
var traceTime int

var sysLoad sysload.SystemLoad
var scanHandler procscan.SchedGroupHandler

var delayTaskCh = make(chan uint64, eventsBufferLen)

func isThreadBind(tid uint64) bool {
	_, ok := bindTaskMap[tid]
	return ok
}

func makeBindGroupByNumaID(numaID int) *bindGroupInfo {
	var bindGroup bindGroupInfo

	bindGroup.numaNode = topology.GetNumaNodeByID(numaID)
	bindGroup.tasks.Init()
	if !isCclAware() {
		bindGroup.bindNode = bindGroup.numaNode
	} else {
		bindGroup.bindNode = bindGroup.numaNode.SelectLighterLoadNode(topology.TopoTypeCluster)
	}
	bindGroup.bindNode.AddBind()
	return &bindGroup
}

func attachTaskToTopoNode(task *bindTaskInfo, node *topology.TopoNode) {
	task.newBindNode = node
	task.nodeEle = node.AttachTask.PushBack(task)
	node.AddBind()
}

func detachTaskFromTopoNode(task *bindTaskInfo, node *topology.TopoNode) {
	task.newBindNode = nil
	node.AttachTask.Remove(task.nodeEle)
	node.SubBind()
}

func setTaskAffinity(tid uint64, node *topology.TopoNode) error {
	var cpus []int

	for cpu := node.Mask().Foreach(-1); cpu != -1; cpu = node.Mask().Foreach(cpu) {
		cpus = append(cpus, cpu)
	}
	log.Info("bind ", tid, " to cpu ", cpus)
	if err := sched.SetAffinity(tid, cpus); err != nil {
		return err
	}
	return nil
}

func bindTaskToTopoNode(taskInfo *bindTaskInfo, node *topology.TopoNode) {
	if err := setTaskAffinity(taskInfo.tid, node); err != nil {
		log.Error(err)
		taskInfo.retryEle = retryTasks.PushBack(taskInfo)
	}
	attachTaskToTopoNode(taskInfo, node)
}

func unbindTaskFromTopoNode(taskInfo *bindTaskInfo, node *topology.TopoNode) {
	detachTaskFromTopoNode(taskInfo, node)
}

func migrateTaskToTopoNode(taskInfo *bindTaskInfo, newNode *topology.TopoNode) {
	taskLoad := sysLoad.GetTaskLoad(taskInfo.tid)
	oldNode := taskInfo.newBindNode
	taskInfo.oldBindNode = oldNode

	unbindTaskFromTopoNode(taskInfo, oldNode)
	oldNode.SubLoad(taskLoad)

	bindTaskToTopoNode(taskInfo, newNode)
	newNode.AddLoad(taskLoad)
}

func attachTaskToGroup(task *bindTaskInfo, group *bindGroupInfo) {
	task.groupEle = group.tasks.PushBack(task)
	task.group = group
}

func detachTaskFromGroup(task *bindTaskInfo, group *bindGroupInfo) {
	group.tasks.Remove(task.groupEle)
	task.group = nil
	task.groupEle = nil
	if group.tasks.Len() == 0 {
		delete(bindGroupMap, group.name)
	}
}

func getTasksNumaAuto(tids []uint64) int {
	var numaFaults [topology.NumaNodeNR]float64
	var maxFaults float64
	var numaID = -1

	for _, tid := range tids {
		topo := topology.GetThreadMemTopo(tid)
		for index, count := range topo.NumaFaults {
			numaFaults[index] += float64(count)
		}
	}
	for i, faults := range numaFaults {
		if faults > maxFaults {
			maxFaults = faults
			numaID = i
		}
	}
	return numaID
}

func bindTasksToNuma(tid uint64, name string, numaID int) {
	var taskInfo bindTaskInfo
	var groupInfo *bindGroupInfo
	var node *topology.TopoNode
	var ok bool

	if groupInfo, ok = bindGroupMap[name]; !ok {
		groupInfo = makeBindGroupByNumaID(numaID)
		bindGroupMap[name] = groupInfo
		groupInfo.name = name
	}

	if isThreadBind(tid) {
		return
	}

	if !isCoarseGrain() {
		node = groupInfo.bindNode.SelectLighterBindNode(topology.TopoTypeCPU)
	} else {
		node = groupInfo.bindNode
	}

	taskInfo.tid = tid
	attachTaskToGroup(&taskInfo, groupInfo)
	bindTaskToTopoNode(&taskInfo, node)
	bindTaskMap[tid] = &taskInfo
	sysLoad.AddTask(tid)
}

// BindGroupAuto bind tasks in group
func BindGroupAuto(tids []uint64, name string) {
	var numaID = -1

	if isNumaAware() {
		numaID = getTasksNumaAuto(tids)
	}

	if isNetAware() {
		netNuma, err := netaffinity.GetProcessNetNuma(tids[0])
		if err != nil {
			log.Info(err)
		} else if netNuma != -1 {
			numaID = netNuma
			log.Debugf("Set net affinity NUMA: %d", numaID)
		}
	}

	if numaID == -1 {
		node := topology.SelectTypeNode(topology.TopoTypeNUMA)
		numaID = node.ID()
	}

	log.Debug("Bind group ", name, tids)
	for _, tid := range tids {
		bindTasksToNuma(tid, name, numaID)
	}
}

func retryBindTasks() {
	for ele := retryTasks.Front(); ele != nil; ele = ele.Next() {
		taskInfo, ok := ele.Value.(*bindTaskInfo)
		if !ok {
			log.Error("get task infomation from failed tasks list failed")
			continue
		}
		if err := setTaskAffinity(taskInfo.tid, taskInfo.newBindNode); err != nil {
			log.Errorf("bind failed task %d failed, ignore task", taskInfo.tid)
			UnbindTaskPolicy(taskInfo.tid)
		}
	}
}

func unbindAllTasks() {
	for ele := retryTasks.Front(); ele != nil; ele = ele.Next() {
		taskInfo, ok := ele.Value.(*bindTaskInfo)
		if !ok {
			log.Error("remove task infomation from failed tasks list failed")
			continue
		}
		retryTasks.Remove(taskInfo.retryEle)
	}
	for tid := range bindTaskMap {
		UnbindTaskPolicy(tid)
	}
}

func autoGetGroups(tidsSlice *[]threadaffinity.TidsGroup, groupNum *int) {
	*tidsSlice, *groupNum = threadaffinity.GetTidSlice()
}

func bindTasksAwareGroup(name string, tracetime int, block *ControlBlock) {
	var tidsSlice []threadaffinity.TidsGroup
	var groupNum int
	log.Debugf("parse process %s affini\n", name)
	if ShouldStartPtraceScan(block) {
		threadaffinity.StartGroups(name, tracetime)
		autoGetGroups(&tidsSlice, &groupNum)
		if threadaffinity.GroupChanged() {
			unbindAllTasks()
			log.Info("group changed, rebind\n")
			for i := 0; i < groupNum; i++ {
				BindGroupAuto(tidsSlice[i].Tids, tidsSlice[i].GroupName)
			}
		}
	}
}

func taskExist(taskName string) {
	if pid, err := utils.GetPid(taskName); err == nil {
		log.Debugf("get %s pid %d\n", taskName, pid)
		delayTaskCh <- pid
		threadaffinity.PidChanged(pid)
	}
}

func bindTasksDelayAwareGroup(taskName string, traceTime int, block *ControlBlock) {
	count := len(delayTaskCh)
	for i := 0; i < count; i++ {
		<-delayTaskCh
		bindTasksAwareGroup(taskName, traceTime, block)
	}
	go taskExist(taskName)
	scanHandler.ScanBoundTask()
}

func initScanHandler() {
	scanHandler.BindGroup = BindGroupAuto
	scanHandler.UnbindTask = UnbindTaskPolicy
	scanHandler.ForEachBoundTask = ForEachBoundTask
}

func delayBindTasks(pid uint64) {
	delayTaskCh <- pid
}

func bindTasksDelay() {
	count := len(delayTaskCh)

	for i := 0; i < count; i++ {
		pid := <-delayTaskCh
		scanHandler.ParseSchedGroupInfo(pid)
	}
	go procscan.ScanEachProc(delayBindTasks)
	scanHandler.ScanBoundTask()
}

func bindTasksDirect() {
	scanHandler.Run()
}

// BindTasksPolicy bind tasks config by user in environment variable
func BindTasksPolicy(block *ControlBlock) {
	retryBindTasks()
	if isAffinityAware() {
		bindTasksDelayAwareGroup(taskName, traceTime, block)
	} else {
		if isNumaAware() {
			bindTasksDelay()
		} else {
			bindTasksDirect()
		}
	}
}

// UnbindTaskPolicy release resouce allocted when bind tasks
func UnbindTaskPolicy(tid uint64) {
	taskInfo, ok := bindTaskMap[tid]
	if !ok {
		return
	}

	log.Info("unbind task ", tid)
	detachTaskFromGroup(taskInfo, taskInfo.group)
	unbindTaskFromTopoNode(taskInfo, taskInfo.newBindNode)
	if taskInfo.retryEle != nil {
		retryTasks.Remove(taskInfo.retryEle)
		taskInfo.retryEle = nil
	}
	delete(bindTaskMap, tid)
	sysLoad.RemoveTask(tid)
}

// ForEachBoundTask foreach all bound tasks with func callback
func ForEachBoundTask(handler func(uint64)) {
	for key := range bindTaskMap {
		handler(key)
	}
}

func cpuGap() int {
	return (cpuGapPct << sysload.ScaleShift) / toPercentage
}

func nodeBalanceGap(node *topology.TopoNode) int {
	return node.Mask().Weight() * cpuGap()
}

func shouldBalanceBetweenNode(srcNode *topology.TopoNode, dstNode *topology.TopoNode) bool {
	diff := srcNode.GetLoad() - dstNode.GetLoad()
	gap := nodeBalanceGap(srcNode)
	return diff > gap
}

func groupTargetNode(srcNode *topology.TopoNode) *topology.TopoNode {
	if srcNode.Type().Compare(topology.TopoTypeNUMA) >= 0 {
		return nil
	}

	numaNode := srcNode.Parent(topology.TopoTypeNUMA)
	return numaNode.SelectLighterLoadNode(srcNode.Type())
}

func groupWeight(group *bindGroupInfo) float64 {
	var weight float64

	for ele := group.tasks.Front(); ele != nil; ele = ele.Next() {
		task, ok := (ele.Value).(*bindTaskInfo)
		if !ok {
			continue
		}
		weight += float64(sysLoad.GetTaskLoad(task.tid))
	}
	return weight
}

func shouldMigrateGroup(group *bindGroupInfo, srcNode *topology.TopoNode, dstNode *topology.TopoNode) bool {
	if !shouldBalanceBetweenNode(srcNode, dstNode) {
		return false
	}

	gWeight := groupWeight(group)
	diff := srcNode.GetLoad() - dstNode.GetLoad()
	return float64(diff) > gWeight
}

func migrateGroup(group *bindGroupInfo, srcNode *topology.TopoNode, dstNode *topology.TopoNode) {
	for ele := group.tasks.Front(); ele != nil; ele = ele.Next() {
		task, ok := (ele.Value).(*bindTaskInfo)
		if !ok {
			continue
		}
		newNode := dstNode.SelectLighterBindNode(task.newBindNode.Type())
		migrateTaskToTopoNode(task, newNode)
	}
	srcNode.SubBind()
	group.bindNode = dstNode
	dstNode.AddBind()
}

func balanceGroup(group *bindGroupInfo) {
	srcNode := group.bindNode
	dstNode := groupTargetNode(srcNode)
	if dstNode == nil {
		return
	}

	if shouldMigrateGroup(group, srcNode, dstNode) {
		migrateGroup(group, srcNode, dstNode)
		return
	}
}

type updateLoadCallback struct{}

// callback to update cpu load
func (callback *updateLoadCallback) Callback(node *topology.TopoNode) {
	cpu := node.ID()
	load := sysLoad.GetCPULoad(cpu)
	node.SetLoad(load)
}

func updateNodeLoad() {
	var callback updateLoadCallback

	sysLoad.Update()
	topology.ForeachTypeCall(topology.TopoTypeCPU, &callback)
}

// BalanceTaskPolicy balance CCLs inside NUMA according cpus' and tasks' load
func BalanceTaskPolicy() {
	updateNodeLoad()

	if !isCclAware() {
		return
	}

	for _, groupInfo := range bindGroupMap {
		balanceGroup(groupInfo)
	}
}

// SwitchNumaAware select NUMA by NUMA faults if on, else by cpu load
func SwitchNumaAware(on bool) {
	numaAwareOn = on
}

func isNumaAware() bool {
	return numaAwareOn
}

// SwitchNetAware select NUMA node according to net affinity
func SwitchNetAware(on bool) {
	netAwareOn = on
}

func isNetAware() bool {
	return netAwareOn
}

// SwitchCclAware choose CCL futher inside NUMA futher if on
func SwitchCclAware(on bool) {
	cclAwareOn = on
}

func isCclAware() bool {
	return cclAwareOn
}

// SwitchCoarseGrain choose cpu futher inside CCL futher if off
func SwitchCoarseGrain(on bool) {
	coarseGrain = on
}

func isCoarseGrain() bool {
	return coarseGrain
}

// SwitchAffinityAware enable detecting thread affinity automaticly
func SwitchAffinityAware(on bool) {
	affinityAware = on
}

func isAffinityAware() bool {
	return affinityAware
}

// SetAffinityTaskName set the target process according to the comm given
func SetAffinityTaskName(name string) {
	taskName = name
}

// SetAffinityTraceTime set the length of tracing time
func SetAffinityTraceTime(time int) {
	traceTime = time
}

// Init init modules we depends on
func Init() error {
	if err := topology.InitTopo(); err != nil {
		log.Error("init topology failed")
		return err
	}
	retryTasks.Init()
	sysLoad.Init()
	initScanHandler()
	return nil
}

// PtraceScanStart start the ptrace scan
func PtraceScanStart(block *ControlBlock) {
	log.Info("threads scan on")
	atomic.StoreInt32(&(block.ptraceScanSwitch), 1)
}

// PtraceScanEnd stop the ptrace scan
func PtraceScanEnd(block *ControlBlock) {
	log.Info("threads scan off")
	atomic.StoreInt32(&(block.ptraceScanSwitch), 0)
}

// ShouldStartPtraceScan indicate whether scanning should be done
func ShouldStartPtraceScan(block *ControlBlock) bool {
	res := atomic.LoadInt32(&(block.ptraceScanSwitch))
	return res == 1
}
