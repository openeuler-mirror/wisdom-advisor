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

// Package topology implements utility to get system topology infomation
package topology

import (
	"container/list"
	"fmt"
	"gitee.com/wisdom-advisor/common/cpumask"
	"io/ioutil"
	"strconv"
	"strings"

	"gitee.com/wisdom-advisor/common/utils"

	log "github.com/sirupsen/logrus"
)

const (
	// NumaNodeNR represents NUMA nodes number
	NumaNodeNR = 4
)

// ThreadMemTopo represents access of NUMA nodes
type ThreadMemTopo struct {
	tid        uint64
	NumaFaults [NumaNodeNR]int64
}

// GetThreadMemTopo get thread memory page relation.
func GetThreadMemTopo(tid uint64) ThreadMemTopo {
	var topo ThreadMemTopo
	topo.tid = tid
	path := fmt.Sprintf(utils.ProcDir+"%d/task_fault_siblings", tid)
	lines := utils.ReadAllFile(path)
	for _, line := range strings.Split(lines, "\n") {
		if !strings.Contains(line, "node") {
			continue
		}
		var node int
		var count int64
		_, err := fmt.Sscanf(line, "faults on node %d: %d", &node, &count)
		if err != nil {
			log.Error(err)
			continue
		}
		if node < NumaNodeNR {
			topo.NumaFaults[node] = count
		}
	}
	return topo
}

const (
	cpuOnlineFile = "online"
	coreMaskFile  = "topology/thread_siblings"
	coreIDFile    = "topology/core_id"
	numaMaskFile  = "cpumap"
	chiMaskFile   = "topology/core_siblings"
	chipIDFile    = "topology/physical_package_id"
)

const coresPerCluster = 4

var sysCPUPath string

// TopoType is topology level of TopoNode
type TopoType uint32

// TopoType ranges
const (
	TopoTypeCPU TopoType = iota
	TopoTypeCore
	TopoTypeCluster
	TopoTypeNUMA
	TopoTypeChip
	TopoTypeAll
	TopoTypeEnd
)

// Compare of TopoType return > 0 if TopoType t contains more cpus than TopoType src
func (t TopoType) Compare(src TopoType) int {
	return int(t) - int(src)
}

type topoLoadInfo struct {
	bindCount int
	load      int
}

// return load as weight
func (loadInfo *topoLoadInfo) Weight() int {
	return loadInfo.load
}

func (loadInfo *topoLoadInfo) compareLoad(src *topoLoadInfo) int {
	return loadInfo.Weight() - src.Weight()
}

func (loadInfo *topoLoadInfo) compareBind(src *topoLoadInfo) int {
	return loadInfo.bindCount - src.bindCount
}

func (loadInfo *topoLoadInfo) add(load int) {
	loadInfo.load += load
}

// Sub do sub to sysload
func (loadInfo *topoLoadInfo) Sub(load int) {
	loadInfo.load -= load
}

// GetLoad can get load info
func (loadInfo *topoLoadInfo) GetLoad() int {
	return loadInfo.load
}

func (loadInfo *topoLoadInfo) addBind() {
	loadInfo.bindCount++
}

func (loadInfo *topoLoadInfo) subBind() {
	loadInfo.bindCount--
}

func (loadInfo *topoLoadInfo) getBind() int {
	return loadInfo.bindCount
}

// TopoTree contains topology infomation of system
type TopoTree struct {
	numaMap  map[int]*TopoNode
	typeList [TopoTypeEnd]list.List
	root     TopoNode
}

var tree TopoTree

// TopoNode is one node in topology tree
type TopoNode struct {
	parent     *TopoNode
	typeEle    *list.Element
	id         int
	topotype   TopoType
	mask       cpumask.Cpumask
	loadInfo   topoLoadInfo
	child      list.List
	AttachTask list.List
}

// Init TopoNode
func (node *TopoNode) Init() {
	node.mask.Init()
	node.parent = nil
	node.child.Init()
	node.AttachTask.Init()
}

func getMaskFromFile(path string) *cpumask.Cpumask {
	var mask cpumask.Cpumask
	line := utils.ReadAllFile(path)
	if line == "" {
		log.Errorf("get mask from %s failed", path)
		return nil
	}
	line = strings.Replace(line, ",", "", -1)
	line = strings.Replace(line, "\n", "", -1)
	err := mask.ParseString(line)
	if err != nil {
		log.Error(err)
		return nil
	}
	return &mask
}

func getIDFromFile(path string) int {
	line := utils.ReadAllFile(path)
	if line == "" {
		log.Errorf("get Id from %s failed", path)
		return -1
	}
	line = strings.Replace(line, "\n", "", -1)
	i, err := strconv.Atoi(line)
	if err != nil {
		log.Errorf("convert Id from %s failed", line)
		return -1
	}
	return i
}

func getChipID(cpu int) int {
	chipIDPath := fmt.Sprintf("%s/cpu%d/%s", sysCPUPath, cpu, chipIDFile)
	return getIDFromFile(chipIDPath)
}

func getChipMask(cpu int) *cpumask.Cpumask {
	chipMaskPath := fmt.Sprintf("%s/cpu%d/%s", sysCPUPath, cpu, chiMaskFile)
	return getMaskFromFile(chipMaskPath)
}

// GetNumaID is to get NUMA ID of specified CPU
func GetNumaID(cpu int) int {
	return getNumaID(cpu)
}

func getNumaID(cpu int) int {
	var node = -1
	var err error
	const nodeIDIndex = 4
	cpuDirPath := fmt.Sprintf("%s/cpu%d/", sysCPUPath, cpu)

	dir, err := ioutil.ReadDir(cpuDirPath)
	if err != nil {
		log.Error(err)
		return -1
	}

	for _, f := range dir {
		if strings.HasPrefix(f.Name(), "node") {
			node, err = strconv.Atoi(f.Name()[nodeIDIndex:])
			if err != nil {
				log.Error(err)
				node = -1
			}
			break
		}
	}
	return node
}

func getNumaMask(cpu int) *cpumask.Cpumask {
	nodeID := getNumaID(cpu)
	if nodeID == -1 {
		log.Errorf("get numa id of cpu %d failed", cpu)
		return nil
	}

	numaMaskPath := fmt.Sprintf("%s/cpu%d/node%d/%s", sysCPUPath, cpu, nodeID, numaMaskFile)
	return getMaskFromFile(numaMaskPath)
}

func getClusterID(cpu int) int {
	return cpu / coresPerCluster
}

func getClusterMask(cpu int) *cpumask.Cpumask {
	var mask cpumask.Cpumask

	mask.Init()
	cpuStart := cpu - cpu%coresPerCluster
	for i := 0; i < coresPerCluster; i++ {
		mask.Set(cpuStart + i)
	}
	return &mask
}

func getCoreID(cpu int) int {
	coreIDPath := fmt.Sprintf("%s/cpu%d/%s", sysCPUPath, cpu, coreIDFile)
	return getIDFromFile(coreIDPath)
}

func getCoreMask(cpu int) *cpumask.Cpumask {
	coreMaskPath := fmt.Sprintf("%s/cpu%d/%s", sysCPUPath, cpu, coreMaskFile)
	return getMaskFromFile(coreMaskPath)
}

func findTypeTopoNode(topotype TopoType, mask *cpumask.Cpumask) *TopoNode {
	typeHead := tree.typeList[topotype]

	for ele := typeHead.Front(); ele != nil; ele = ele.Next() {
		node, ok := (ele.Value).(*TopoNode)
		if !ok {
			return nil
		}
		if node.mask.IsEqual(mask) {
			return node
		}
	}
	return nil
}

func getChipNode(cpu int) *TopoNode {
	var node TopoNode

	chipMask := getChipMask(cpu)
	if chipMask == nil {
		log.Errorf("get chip mask for cpu %d failed", cpu)
		return nil
	}

	existNode := findTypeTopoNode(TopoTypeChip, chipMask)
	if existNode != nil {
		return existNode
	}

	node.Init()
	node.mask.Copy(chipMask)
	node.id = getChipID(cpu)
	if node.id == -1 {
		log.Errorf("get chip id for cpu %d failed", cpu)
		return nil
	}
	node.topotype = TopoTypeChip
	return &node
}

func getNumaNode(cpu int) *TopoNode {
	var node TopoNode

	numaMask := getNumaMask(cpu)
	if numaMask == nil {
		log.Errorf("get numa mask for cpu %d failed", cpu)
		return nil
	}
	existNode := findTypeTopoNode(TopoTypeNUMA, numaMask)
	if existNode != nil {
		return existNode
	}

	node.Init()
	node.mask.Copy(numaMask)
	node.id = getNumaID(cpu)
	if node.id == -1 {
		log.Errorf("get numa id for cpu %d failed", cpu)
		return nil
	}
	node.topotype = TopoTypeNUMA
	tree.numaMap[node.id] = &node
	return &node
}

func getClusterNode(cpu int) *TopoNode {
	var node TopoNode

	clusterMask := getClusterMask(cpu)
	existNode := findTypeTopoNode(TopoTypeCluster, clusterMask)
	if existNode != nil {
		return existNode
	}

	node.Init()
	node.mask.Copy(clusterMask)
	node.id = getClusterID(cpu)
	node.topotype = TopoTypeCluster
	return &node
}

func getCoreNode(cpu int) *TopoNode {
	var node TopoNode

	coreMask := getCoreMask(cpu)
	if coreMask == nil {
		log.Errorf("get core mask of cpu %d failed", cpu)
		return nil
	}
	existNode := findTypeTopoNode(TopoTypeCore, coreMask)
	if existNode != nil {
		return existNode
	}

	node.Init()
	node.mask.Copy(coreMask)
	node.id = getCoreID(cpu)
	if node.id == -1 {
		log.Errorf("get core id of cpu %d failed", cpu)
		return nil
	}
	node.topotype = TopoTypeCore
	return &node
}

func getCPUNode(cpu int) *TopoNode {
	var node TopoNode

	node.Init()
	node.mask.Set(cpu)
	node.id = cpu
	node.topotype = TopoTypeCPU
	return &node
}

func isCPUOnline(cpu int) bool {
	onlinePath := fmt.Sprintf("%s/cpu%d/%s", sysCPUPath, cpu, cpuOnlineFile)
	line := utils.ReadAllFile(onlinePath)
	line = strings.Replace(line, "\n", "", -1)
	isOnline, err := strconv.Atoi(line)
	if err != nil {
		log.Error("get cpu online failed ", err)
		return false
	}
	return isOnline > 0
}

// CallBack to foreach topology nodes
type CallBack interface {
	Callback(node *TopoNode)
}

func (node *TopoNode) foreachChildCall(fun CallBack) {
	fun.Callback(node)
	for ele := node.child.Front(); ele != nil; ele = ele.Next() {
		(ele.Value).(*TopoNode).foreachChildCall(fun)
	}
}

type lighterBindCallback struct {
	node *TopoNode
	t    TopoType
}

// Callback to find less bind node
func (callback *lighterBindCallback) Callback(node *TopoNode) {
	if node.topotype != callback.t {
		return
	}
	if callback.node == nil {
		callback.node = node
		return
	}
	if node.loadInfo.compareBind(&callback.node.loadInfo) < 0 {
		callback.node = node
		return
	}
}

// SelectLighterBindNode find the least bind child node with t TopoType
func (node *TopoNode) SelectLighterBindNode(t TopoType) *TopoNode {
	var callback lighterBindCallback
	callback.t = t
	callback.node = nil

	node.foreachChildCall(&callback)
	return callback.node
}

type lighterLoadCallback struct {
	node *TopoNode
	t    TopoType
}

// Callback to find less load node
func (callback *lighterLoadCallback) Callback(node *TopoNode) {
	if node.topotype != callback.t {
		return
	}
	if callback.node == nil {
		callback.node = node
		return
	}
	diff := node.loadInfo.compareLoad(&callback.node.loadInfo)
	if diff < 0 || (diff == 0 && node.loadInfo.compareBind(&callback.node.loadInfo) < 0) {
		callback.node = node
	}
}

// SelectLighterLoadNode find the least load child node with t TopoType
func (node *TopoNode) SelectLighterLoadNode(t TopoType) *TopoNode {
	var callback lighterLoadCallback
	callback.t = t
	callback.node = nil

	node.foreachChildCall(&callback)
	return callback.node
}

// Parent return parent TopoNode
func (node *TopoNode) Parent(t TopoType) *TopoNode {
	var p *TopoNode

	if t.Compare(node.topotype) < 0 {
		return nil
	}

	for p = node; p != nil && p.topotype != t; p = p.parent {
		// NULL
	}
	return p
}

type loadChangeCallback struct {
	load int
}

// Callback to update node's load
func (callback *loadChangeCallback) Callback(node *TopoNode) {
	if node.topotype == TopoTypeCPU {
		node.AddLoad(callback.load)
	}
}

// AddLoad add load to TopoNode
func (node *TopoNode) AddLoad(load int) {
	if node.topotype == TopoTypeCPU {
		p := node
		for p != nil {
			p.loadInfo.add(load)
			p = p.parent
		}
	} else {
		var callback loadChangeCallback
		callback.load = load / node.mask.Weight()
		node.foreachChildCall(&callback)
	}
}

// SubLoad sub load from TopoNode
func (node *TopoNode) SubLoad(load int) {
	node.AddLoad(-load)
}

// SetLoad set load of TopoNode
func (node *TopoNode) SetLoad(load int) {
	node.AddLoad(load - node.GetLoad())
}

// GetLoad get load from TopoNode
func (node *TopoNode) GetLoad() int {
	return node.loadInfo.GetLoad()
}

// AddBind add bind number of TopoNode
func (node *TopoNode) AddBind() {
	node.loadInfo.addBind()
}

// SubBind sub bind number of TopoNode
func (node *TopoNode) SubBind() {
	node.loadInfo.subBind()
}

// GetBind get bind number of TopoNode
func (node *TopoNode) GetBind() int {
	return node.loadInfo.getBind()
}

// ID return id of TopoNode
func (node *TopoNode) ID() int {
	return node.id
}

// Type return TopoType of TopoNode
func (node *TopoNode) Type() TopoType {
	return node.topotype
}

// Mask return cpumask of TopoNode
func (node *TopoNode) Mask() *cpumask.Cpumask {
	return &node.mask
}

// GetNumaNodeByID return NUMA TopoNode with id
func GetNumaNodeByID(id int) *TopoNode {
	return tree.numaMap[id]
}

// InitTopo get topology infomation from sysfs
func InitTopo() error {
	for i := range tree.typeList {
		tree.typeList[i].Init()
	}

	tree.root.Init()
	tree.root.topotype = TopoTypeAll
	tree.numaMap = make(map[int]*TopoNode)

	sysCPUPath = fmt.Sprintf("%s/devices/system/cpu/", utils.SysDir)
	for cpu := 0; cpu < utils.CPUNum; cpu++ {
		if !isCPUOnline(cpu) {
			log.Infof("cpu %d offline, ignore", cpu)
			continue
		}
		chipNode := getChipNode(cpu)
		if chipNode == nil {
			return fmt.Errorf("get chip node of cpu %d failed", cpu)
		}
		if chipNode.parent == nil {
			chipNode.parent = &tree.root
			tree.root.child.PushBack(chipNode)
			chipNode.typeEle = tree.typeList[TopoTypeChip].PushBack(chipNode)
		}
		numaNode := getNumaNode(cpu)
		if numaNode == nil {
			return fmt.Errorf("get numa node of cpu %d failed", cpu)
		}
		if numaNode.parent == nil {
			numaNode.parent = chipNode
			chipNode.child.PushBack(numaNode)
			numaNode.typeEle = tree.typeList[TopoTypeNUMA].PushBack(numaNode)
		}
		clusterNode := getClusterNode(cpu)
		if clusterNode == nil {
			return fmt.Errorf("get cluster node of cpu %d failed", cpu)
		}
		if clusterNode.parent == nil {
			clusterNode.parent = numaNode
			numaNode.child.PushBack(clusterNode)
			clusterNode.typeEle = tree.typeList[TopoTypeCluster].PushBack(clusterNode)
		}
		coreNode := getCoreNode(cpu)
		if coreNode == nil {
			return fmt.Errorf("get core node of cpu %d failed", cpu)
		}
		if coreNode.parent == nil {
			coreNode.parent = clusterNode
			clusterNode.child.PushBack(coreNode)
			coreNode.typeEle = tree.typeList[TopoTypeCore].PushBack(coreNode)
		}
		cpuNode := getCPUNode(cpu)
		if cpuNode.parent == nil {
			cpuNode.parent = coreNode
			coreNode.child.PushBack(cpuNode)
			cpuNode.typeEle = tree.typeList[TopoTypeCPU].PushBack(cpuNode)
		}
	}
	return nil
}

// ForeachTypeCall foreach all nodes with TopoType t and call fun callback
func ForeachTypeCall(t TopoType, fun CallBack) {
	head := tree.typeList[t]

	for ele := head.Front(); ele != nil; ele = ele.Next() {
		fun.Callback((ele.Value).(*TopoNode))
	}
}

// SelectTypeNode select most less load with TopoType t
func SelectTypeNode(t TopoType) *TopoNode {
	var callback lighterLoadCallback
	callback.t = t
	callback.node = nil

	ForeachTypeCall(t, &callback)
	return callback.node
}
