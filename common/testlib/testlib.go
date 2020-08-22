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

// Package testlib provides methods for testing, basicly to build fake sysfs path
package testlib

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"gitee.com/wisdom-advisor/common/cpumask"
	"gitee.com/wisdom-advisor/common/sched"
	"gitee.com/wisdom-advisor/common/utils"
)

// #include <unistd.h>
import "C"

const defaultPerm = 0644
const testChpNum = 1
const initMapLen = 16

// ConvertPct is to calculate percentage
const ConvertPct = 100
const taskStatFilesNum = 52
const userFiled = 13
const systemFiled = 14
const online = 1

var numaPerChip = 2
var clusterPerNuma = 8
var corePerCluster = 4
var cpuPerCore = 1
var procDir = "./tmp/proc/"
var sysDir = "./tmp/sys/"

var clockTicks int

var bindMap map[uint64][]int

func init() {
	rand.Seed(time.Now().Unix())
	clockTicks = int(C.sysconf(C._SC_CLK_TCK))
}

func pathExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

// BuildFakePathWithData is to build file and parent directory
func BuildFakePathWithData(path string, file string, data []byte) {
	BuildFakePath(path)
	AddFakeDataToPath(path, file, data)
}

// BuildFakePath is to build a fake directory
func BuildFakePath(path string) {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		fmt.Println("Mkdir error")
		return
	}
}

// AddFakeDataToPath is to add file to fake directory
func AddFakeDataToPath(path string, file string, data []byte) {
	err := ioutil.WriteFile(path+file, data, defaultPerm)
	if err != nil {
		fmt.Printf("Write file error")
		return
	}
}

// ExecCommand is to execute a cmd in a string
func ExecCommand(comand string) {
	var out bytes.Buffer
	cmd := exec.Command("/bin/bash", "-c", comand)
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println("Create thread error.")
		return
	}
}

// KillThread is to kill a thread ny pid
func KillThread(pid uint64) {
	comand := fmt.Sprintf("kill -9 %d", pid)
	ExecCommand(comand)
}

// CPUProcStat include fake /proc/stat content
type CPUProcStat struct {
	CPUNum    uint64
	User      uint64
	nice      uint64
	System    uint64
	idle      uint64
	iowait    uint64
	irq       uint64
	softirq   uint64
	steal     uint64
	guest     uint64
	guestNice uint64
}

// Init is to init /proc/stat fake content
func (stat *CPUProcStat) Init(cpu uint64) {
	stat.CPUNum = cpu
	stat.User = uint64(rand.Int63())
	stat.nice = uint64(rand.Int63())
	stat.System = uint64(rand.Int63())
	stat.idle = uint64(rand.Int63())
	stat.iowait = uint64(rand.Int63())
	stat.irq = uint64(rand.Int63())
	stat.softirq = uint64(rand.Int63())
	stat.steal = uint64(rand.Int63())
	stat.guest = uint64(rand.Int63())
	stat.guestNice = uint64(rand.Int63())
}

// AddUser add user ticks
func (stat *CPUProcStat) AddUser(ticks uint64) {
	stat.User += ticks
}

// ToString convert CPUProcStat to string format in /proc/stat
func (stat *CPUProcStat) ToString() string {
	str := fmt.Sprintf(
		"cpu%d  %d %d %d %d %d %d %d %d %d %d\n",
		stat.CPUNum, stat.User, stat.nice, stat.System, stat.idle, stat.iowait,
		stat.irq, stat.softirq, stat.steal, stat.guest, stat.guestNice,
	)
	return str
}

// CPULoadStub is stub of cpu load info
type CPULoadStub struct {
	cpusLoad []CPUProcStat
	procDir  string
}

// NewCPULoadStub init sbub of cpu load info
func NewCPULoadStub(cpuNum int, procDir string) *CPULoadStub {
	var stub CPULoadStub

	if !pathExist(procDir) {
		BuildFakePath(procDir)
	}
	stub.procDir = procDir
	stub.cpusLoad = make([]CPUProcStat, cpuNum)
	for i := 0; i < cpuNum; i++ {
		stub.cpusLoad[i].Init(uint64(i))
	}
	stub.DeployLoad()
	return &stub
}

// AddLoad add cpu load to stub
func (stub *CPULoadStub) AddLoad(cpu int, cpuUsage int, duartion int) {
	ticks := clockTicks * cpuUsage * duartion / ConvertPct
	stub.cpusLoad[cpu].AddUser(uint64(ticks))
}

// DeployLoad write load info to fake file
func (stub *CPULoadStub) DeployLoad() {
	var data string

	for i := 0; i < len(stub.cpusLoad); i++ {
		data += stub.cpusLoad[i].ToString()
	}
	AddFakeDataToPath(stub.procDir, "stat", []byte(data))
}

// CPUToCore return core id of cpu
func CPUToCore(cpu int) int {
	return cpu / cpuPerCore
}

// CPUToCluster return cluster id of cpu
func CPUToCluster(cpu int) int {
	return cpu / (cpuPerCore * corePerCluster)
}

// CPUToNuma return numa id of cpu
func CPUToNuma(cpu int) int {
	return cpu / (cpuPerCore * corePerCluster * clusterPerNuma)
}

// CPUToChip return chip id of cpu
func CPUToChip(cpu int) int {
	return cpu / (cpuPerCore * corePerCluster * clusterPerNuma * numaPerChip)
}

func maskToByte(mask cpumask.Cpumask) []byte {
	var data string
	printCount := 0

	for i := len(mask.Masks) - 1; i >= 0; i-- {
		if mask.Masks[i] == 0 && printCount == 0 {
			continue
		}
		data += fmt.Sprintf("%016x", mask.Masks[i])
		printCount++
	}
	if printCount == 0 {
		data = fmt.Sprintf("%d", 0)
	}
	return []byte(data)
}

// TopoStub is stub of cpu topology info in sysfs
type TopoStub struct {
	sysDir      string
	ChipNum     int
	NumaNum     int
	ClusterNum  int
	CoreNum     int
	CPUNum      int
	chipMask    []cpumask.Cpumask
	numaMask    []cpumask.Cpumask
	clusterMask []cpumask.Cpumask
	coreMask    []cpumask.Cpumask
}

func initOneLevelMask(masks []cpumask.Cpumask, cpuNumPerEntry int) {
	num := len(masks)

	for i := 0; i < num; i++ {
		start := i * cpuNumPerEntry
		end := start + cpuNumPerEntry
		for cpu := start; cpu < end; cpu++ {
			masks[i].Set(cpu)
		}
	}
}

func (stub *TopoStub) initTopoMask() {
	stub.chipMask = make([]cpumask.Cpumask, stub.ChipNum)
	stub.numaMask = make([]cpumask.Cpumask, stub.NumaNum)
	stub.clusterMask = make([]cpumask.Cpumask, stub.ClusterNum)
	stub.coreMask = make([]cpumask.Cpumask, stub.CoreNum)

	cpuNumPerEntry := 1
	cpuNumPerEntry *= cpuPerCore
	initOneLevelMask(stub.coreMask, cpuNumPerEntry)
	cpuNumPerEntry *= corePerCluster
	initOneLevelMask(stub.clusterMask, cpuNumPerEntry)
	cpuNumPerEntry *= clusterPerNuma
	initOneLevelMask(stub.numaMask, cpuNumPerEntry)
	cpuNumPerEntry *= numaPerChip
	initOneLevelMask(stub.chipMask, cpuNumPerEntry)
}

func (stub *TopoStub) initTopoNum(chipNum int) {
	stub.ChipNum = chipNum
	stub.NumaNum = stub.ChipNum * numaPerChip
	stub.ClusterNum = stub.NumaNum * clusterPerNuma
	stub.CoreNum = stub.ClusterNum * corePerCluster
	stub.CPUNum = stub.CoreNum * cpuPerCore
}

func (stub *TopoStub) buildCPUTopo(cpu int) {
	coreID := CPUToCore(cpu)
	numaID := CPUToNuma(cpu)
	chipID := CPUToChip(cpu)

	cpuPath := fmt.Sprintf("%s/devices/system/cpu/cpu%d/", stub.sysDir, cpu)
	cpuTopoPath := fmt.Sprintf("%s/topology/", cpuPath)
	cpuNodePath := fmt.Sprintf("%s/node%d/", cpuPath, numaID)
	BuildFakePath(cpuTopoPath)
	BuildFakePath(cpuNodePath)

	AddFakeDataToPath(cpuPath, "online", []byte(fmt.Sprintf("%d", online)))
	AddFakeDataToPath(cpuTopoPath, "core_siblings",
		maskToByte(stub.chipMask[chipID]))
	AddFakeDataToPath(cpuTopoPath, "physical_package_id",
		[]byte(fmt.Sprintf("%d", chipID)))
	AddFakeDataToPath(cpuNodePath, "cpumap",
		maskToByte(stub.numaMask[numaID]))
	AddFakeDataToPath(cpuTopoPath, "thread_siblings",
		maskToByte(stub.coreMask[coreID]))
	AddFakeDataToPath(cpuTopoPath, "core_id",
		[]byte(fmt.Sprintf("%d", coreID)))
}

// NewTopoStub build a new stub of cpu topology
func NewTopoStub(chipNum int, sysDir string) *TopoStub {
	var stub TopoStub

	stub.sysDir = sysDir
	stub.initTopoNum(chipNum)
	stub.initTopoMask()

	for i := 0; i < stub.CPUNum; i++ {
		stub.buildCPUTopo(i)
	}
	return &stub
}

// TaskProcStat is fake content of /proc/pid/task/tid/stat
type TaskProcStat struct {
	statFiles [taskStatFilesNum]string
	procDir   string
	pid       uint64
	tid       uint64
	user      uint64
	system    uint64
}

// ToString convert content to string format in /proc/pid/task/tid/stat
func (stat *TaskProcStat) ToString() string {
	var data string

	for i := 0; i < taskStatFilesNum; i++ {
		data += fmt.Sprintf("%s ", stat.statFiles[i])
	}
	return data
}

// AddLoad add load to task
func (stat *TaskProcStat) AddLoad(cpuUsage int, duration int) {
	ticks := clockTicks * cpuUsage * duration / ConvertPct
	stat.user += uint64(ticks)
	stat.statFiles[userFiled] = fmt.Sprintf("%d", stat.user)
}

// DeployLoad write load to fake file
func (stat *TaskProcStat) DeployLoad() {
	threadPath := fmt.Sprintf(stat.procDir+"%d/task/%d/", stat.pid, stat.tid)
	AddFakeDataToPath(threadPath, "stat", []byte(stat.ToString()))
}

// NewTaskProcStat return a new task load info
func NewTaskProcStat(pid uint64, tid uint64) *TaskProcStat {
	var stat TaskProcStat
	stat.pid = pid
	stat.tid = tid
	stat.procDir = procDir
	stat.user = uint64(rand.Int63())
	stat.system = uint64(rand.Int63())

	for i := 0; i < taskStatFilesNum; i++ {
		stat.statFiles[i] = "init"
	}
	stat.statFiles[userFiled] = fmt.Sprintf("%d", stat.user)
	stat.statFiles[systemFiled] = fmt.Sprintf("%d", stat.system)
	threadPath := fmt.Sprintf(stat.procDir+"%d/task/%d/", pid, tid)
	if !pathExist(threadPath) {
		BuildFakePath(threadPath)
	}
	AddFakeDataToPath(threadPath, "stat", []byte(stat.ToString()))
	threadPath = fmt.Sprintf(stat.procDir+"%d/", tid)
	if !pathExist(threadPath) {
		BuildFakePath(threadPath)
	}
	AddFakeDataToPath(threadPath, "status", []byte(fmt.Sprintf("Tgid:   %d\n", pid)))
	return &stat
}

// TaskStub create fake task
type TaskStub struct {
	tasksStat map[uint64]*TaskProcStat
	procDir   string
	idr       uint64
	existTids map[uint64]bool
}

// NewTaskStub return a new TaskStub
func NewTaskStub(procDir string) *TaskStub {
	var stub TaskStub
	stub.procDir = procDir
	stub.existTids = make(map[uint64]bool, initMapLen)
	stub.tasksStat = make(map[uint64]*TaskProcStat, initMapLen)
	return &stub
}

func (stub *TaskStub) deployTasks(tids []uint64) {
	mainPid := tids[0]
	env := fmt.Sprintf("__SCHED_GROUP__%d=thread%d", tids[0], tids[0])
	for i := 1; i < len(tids); i++ {
		env += fmt.Sprintf(",thread%d", tids[i])
	}
	mainProcPath := fmt.Sprintf("%s%d/", stub.procDir, mainPid)
	BuildFakePathWithData(mainProcPath, "environ", []byte(env))

	for i := 0; i < len(tids); i++ {
		threadName := fmt.Sprintf("thread%d", tids[i])
		threadProcPath := fmt.Sprintf("%s/%d/task/%d/", stub.procDir, mainPid, tids[i])
		BuildFakePathWithData(threadProcPath, "comm", []byte(threadName))
		threadExistPath := fmt.Sprintf("%s/%d/", stub.procDir, tids[i])
		BuildFakePath(threadExistPath)
	}

	for i := 0; i < len(tids); i++ {
		stub.tasksStat[tids[i]] = NewTaskProcStat(tids[0], tids[i])
	}
}

// CreateTasks create target process with num threads
func (stub *TaskStub) CreateTasks(num int) []uint64 {
	tids := make([]uint64, num, num)
	for i := 0; i < num; i++ {
		tids[i] = stub.idr
		stub.idr++
		stub.existTids[tids[i]] = true
	}
	stub.deployTasks(tids)
	return tids
}

// DeleteTid delete thread data
func (stub *TaskStub) DeleteTid(tid uint64) {
	threadProcPath := fmt.Sprintf("%s/%d/task/%d/", stub.procDir, tid, tid)
	err := os.RemoveAll(threadProcPath)
	if err != nil {
		fmt.Printf("remove %s failed\n", threadProcPath)
	}

	threadExistPath := fmt.Sprintf("%s/%d/", stub.procDir, tid)
	err = os.RemoveAll(threadExistPath)
	if err != nil {
		fmt.Printf("remove %s failed\n", threadExistPath)
	}
}

// AddLoad add load for task with tid in stub
func (stub *TaskStub) AddLoad(tid uint64, cpuUsage int, duration int) {
	taskStat, ok := stub.tasksStat[tid]
	if !ok {
		return
	}
	taskStat.AddLoad(cpuUsage, duration)
	taskStat.DeployLoad()
}

// InitStub create Topology, cpu load and task stub
func InitStub() (*TopoStub, *CPULoadStub, *TaskStub) {
	sched.SetAffinity = SetAffinityStub
	bindMap = make(map[uint64][]int, initMapLen)

	utils.SysDir = sysDir
	topoStub := NewTopoStub(testChpNum, sysDir)

	utils.CPUNum = topoStub.CPUNum
	utils.ProcDir = procDir
	cpuLoadStub := NewCPULoadStub(topoStub.CPUNum, procDir)

	taskStub := NewTaskStub(procDir)

	return topoStub, cpuLoadStub, taskStub
}

// CleanStub clean stub data
func CleanStub() {
	err := os.RemoveAll("./tmp/")
	if err != nil {
		fmt.Printf("Clean Stub failed")
	}
}

// SetAffinityStub is SetAffinity Stub
func SetAffinityStub(tid uint64, cpu []int) error {
	bindMap[tid] = cpu
	return nil
}

// GetAffinityStub get affinity set by SetAffinityStub
func GetAffinityStub(tid uint64) ([]int, error) {
	cpu, ok := bindMap[tid]
	if !ok {
		return nil, fmt.Errorf("task not exist")
	}
	return cpu, nil
}
