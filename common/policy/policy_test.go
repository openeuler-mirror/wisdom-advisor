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

package policy

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"gitee.com/wisdom-advisor/common/sched"
	"gitee.com/wisdom-advisor/common/testlib"
	"gitee.com/wisdom-advisor/common/utils"
)

const nodeNum = 4
const threadNum = 2
const threadLimit = 6000
const faultsLimit = 500
const testCount = 1000
const uintMin = 0
const allSame = -1
const chipNum = 1
const percent100 = 100
const testSleepTime = 1
const nodeCPUNum = 4
const testThreadNum = 1
const groupThreadNum = 4
const coarseGrainThreadNum = 1
const noMigrateGap = 20
const migrateGap = 40
const numerousTaskNum = 500
const acceptTime = 1

func getMaxIndex(arr []uint64) int {
	var max uint64 = uintMin
	ret := allSame
	for i, elem := range arr {
		if elem > max {
			max = elem
			ret = i
		}
	}
	return ret
}

func addCclLoad(cclID int, cpuNumPerCluster int, cpuLoadStub *testlib.CPULoadStub, load int) {
	for i := 0; i < cpuNumPerCluster; i++ {
		cpuLoadStub.AddLoad(cclID*cpuNumPerCluster+i, load, testSleepTime)
	}
	cpuLoadStub.DeployLoad()
}

// TestGetNumaDependOnFaults is random test for getNumaDependOnFaults
func TestGetNumaDependOnFaults(t *testing.T) {
	var numaID int
	utils.ProcDir = "./tmp/proc/"
	tidsMap := make(map[uint64]uint64, threadNum)
	var tids []uint64
	var faultsNum [nodeNum]uint64
	var tmp uint64
	var data []byte
	var path string
	var expect int

	rand.Seed(time.Now().Unix())

	for len(tidsMap) < threadNum {
		tmp = uint64(rand.Int63n(threadLimit))
		tidsMap[tmp] = tmp
	}

	for _, tid := range tidsMap {
		tids = append(tids, tid)
		data = data[:0]
		for i := 0; i < nodeNum; i++ {
			tmp = uint64(rand.Int63n(faultsLimit))
			num := fmt.Sprintf("faults on node %d: %d\n", i, tmp)
			faultsNum[i] = faultsNum[i] + tmp
			data = append(data, []byte(num)...)
		}
		path = fmt.Sprintf("./tmp/proc/%d/", tid)
		testlib.BuildFakePathWithData(path, "task_fault_siblings", data)
	}
	expect = getMaxIndex(faultsNum[:])
	if expect == allSame {
		return
	}
	numaID = getTasksNumaAuto(tids)

	err := os.RemoveAll("./tmp/")
	if err != nil {
		fmt.Println("Remove path error")
	}

	if numaID != expect {
		t.Errorf(" Select %d; expected %d", numaID, expect)
	}
}

// TestAll is multiple test for getNumaDependOnFaults
func TestAll(t *testing.T) {
	for i := 0; i < testCount; i++ {
		t.Run("TestGetNumaDependOnFaults", TestGetNumaDependOnFaults)
	}
}

func checkBind(tids []uint64, targetCcl int, cpuNumPerCluster int, t *testing.T) {
	tidNum := len(tids)
	cpuRec := make(map[uint64]bool, tidNum)

	for _, tid := range tids {
		cpus, err := testlib.GetAffinityStub(tid)
		if err != nil {
			t.Errorf("get affinity failed\n")
		}
		ccl := cpus[0] / cpuNumPerCluster
		if ccl != targetCcl {
			t.Errorf("bind ccl wrong, expect %d, actually %d\n",
				targetCcl, ccl)
		}
		if _, ok := cpuRec[tid]; ok {
			t.Errorf("cpu already bind")
		} else {
			cpuRec[tid] = true
		}
	}
}

// func TestBindTaskPolicy test BindTaskPolicy
func TestBindTaskPolicy(t *testing.T) {
	var block ControlBlock

	topoStub, cpuLoadStub, taskStub := testlib.InitStub()
	tids := taskStub.CreateTasks(groupThreadNum)
	cpuNumPerCluster := topoStub.CPUNum / topoStub.ClusterNum
	SwitchCclAware(true)
	if err := Init(); err != nil {
		t.Errorf("init policy failed\n")
	}
	targetCcl := rand.Intn(topoStub.ClusterNum)
	for i := 0; i < topoStub.ClusterNum; i++ {
		if i != targetCcl {
			addCclLoad(i, cpuNumPerCluster, cpuLoadStub, percent100)
		}
	}

	time.Sleep(time.Duration(testSleepTime) * time.Second)
	BalanceTaskPolicy()
	BindTasksPolicy(&block)

	checkBind(tids, targetCcl, cpuNumPerCluster, t)

	for i := 0; i < len(tids); i++ {
		UnbindTaskPolicy(tids[i])
	}
	SwitchCclAware(false)
	testlib.CleanStub()
}

// TestDelayBindTasks test delayBindTasks
func TestDelayBindTasks(t *testing.T) {
	var block ControlBlock
	_, _, taskStub := testlib.InitStub()
	SwitchNumaAware(true)
	tids := taskStub.CreateTasks(testThreadNum)

	if err := Init(); err != nil {
		t.Errorf("init policy failed\n")
	}
	// identify task for delay bind
	BindTasksPolicy(&block)
	if _, err := testlib.GetAffinityStub(tids[0]); err == nil {
		t.Errorf("bind task early\n")
	}
	time.Sleep(time.Duration(testSleepTime) * time.Second)
	// delay bind task
	BindTasksPolicy(&block)
	if _, err := testlib.GetAffinityStub(tids[0]); err != nil {
		t.Errorf("get affinity failed %s\n", err.Error())
	}
	UnbindTaskPolicy(tids[0])
	SwitchNumaAware(false)
	testlib.CleanStub()
}

func setAffinityFailed(tid uint64, cpu []int) error {
	return fmt.Errorf("setAffinityFailed stub")
}

func normalRetryTest(tid uint64, t *testing.T) {
	var block ControlBlock
	// set affinity failed when bind task
	sched.SetAffinity = setAffinityFailed
	BindTasksPolicy(&block)
	// retry failed task, set affinity sucess
	sched.SetAffinity = testlib.SetAffinityStub
	BindTasksPolicy(&block)

	if _, err := testlib.GetAffinityStub(tid); err != nil {
		t.Error(err.Error())
	}
	// clean up tid bind info
	UnbindTaskPolicy(tid)
}

func taskExitBeforeRetryTest(tid uint64, taskStub *testlib.TaskStub, t *testing.T) {
	var block ControlBlock
	// set affinity failed
	sched.SetAffinity = setAffinityFailed
	BindTasksPolicy(&block)
	// failed task exit
	taskStub.DeleteTid(tid)
	// retry failed task, pass if no coredump
	BindTasksPolicy(&block)
	// release resource as retry failed
	if isThreadBind(tid) {
		t.Errorf("thread info still exist after retry failed\n")
	}
}

// TestRetryBindTasks is a test for retryBindTasks function
func TestRetryBindTasks(t *testing.T) {
	_, _, taskStub := testlib.InitStub()

	tids := taskStub.CreateTasks(testThreadNum)
	if err := Init(); err != nil {
		t.Errorf("policy Init failed\n")
	}

	normalRetryTest(tids[0], t)
	taskExitBeforeRetryTest(tids[0], taskStub, t)

	testlib.CleanStub()
}

// TestUnbindTaskPolicy test UnbindTaskPolicy
func TestUnbindTaskPolicy(t *testing.T) {
	var block ControlBlock
	_, _, taskStub := testlib.InitStub()
	tids := taskStub.CreateTasks(testThreadNum)
	if err := Init(); err != nil {
		t.Errorf("init policy failed\n")
	}

	BindTasksPolicy(&block)
	if !isThreadBind(tids[0]) {
		t.Errorf("task bind failed\n")
	}

	taskStub.DeleteTid(tids[0])
	BindTasksPolicy(&block)
	if isThreadBind(tids[0]) {
		t.Errorf("task unbind failed\n")
	}
	testlib.CleanStub()
}

// TestMigration test Migration mechanism
func TestMigration(t *testing.T) {
	var block ControlBlock
	topoStub, cpuLoadStub, taskStub := testlib.InitStub()
	SwitchCclAware(true)
	if err := Init(); err != nil {
		t.Error("init policy failed")
	}
	tids := taskStub.CreateTasks(testThreadNum)
	cpuNumPerCluster := topoStub.CPUNum / topoStub.ClusterNum
	srcCcl := rand.Intn(topoStub.ClusterNum)
	for i := 0; i < topoStub.ClusterNum; i++ {
		if i != srcCcl {
			addCclLoad(i, cpuNumPerCluster, cpuLoadStub, percent100)
		}
	}
	time.Sleep(time.Duration(testSleepTime) * time.Second)
	BalanceTaskPolicy()
	BindTasksPolicy(&block)

	cclNumPerNuma := topoStub.ClusterNum / topoStub.NumaNum
	// make sure dstCcl and srcCcl in same NUMA
	dstCcl := srcCcl/cclNumPerNuma*cclNumPerNuma +
		(srcCcl+1)%cclNumPerNuma
	for i := 0; i < topoStub.ClusterNum; i++ {
		if i != dstCcl {
			addCclLoad(i, cpuNumPerCluster, cpuLoadStub, percent100)
		}
	}
	time.Sleep(time.Duration(testSleepTime) * time.Second)
	BalanceTaskPolicy()

	cpus, err := testlib.GetAffinityStub(tids[0])
	if err != nil {
		t.Error(err.Error())
	}
	if cpus[0]/cpuNumPerCluster != dstCcl {
		t.Errorf("migrate failed\n")
	}
	UnbindTaskPolicy(tids[0])
	SwitchCclAware(false)
	testlib.CleanStub()
}

func getBindCcl(tid uint64, cpuNumPerCluster int, t *testing.T) int {
	cpus, err := testlib.GetAffinityStub(tid)
	if err != nil {
		t.Errorf("get affinity failed")
		return -1
	}
	return cpus[0] / cpuNumPerCluster
}

// TestMigrationGap test migration gap
func TestMigrationWithGap(t *testing.T) {
	var block ControlBlock
	topoStub, cpuLoadStub, taskStub := testlib.InitStub()
	SwitchCclAware(true)
	if err := Init(); err != nil {
		t.Error("init policy failed")
	}
	tids := taskStub.CreateTasks(testThreadNum)
	cpuNumPerCluster := topoStub.CPUNum / topoStub.ClusterNum

	// bind a init ccl
	BindTasksPolicy(&block)
	srcTarget := getBindCcl(tids[0], cpuNumPerCluster, t)

	// test load within migration gap
	addCclLoad(srcTarget, cpuNumPerCluster, cpuLoadStub, noMigrateGap)
	time.Sleep(time.Duration(testSleepTime) * time.Second)
	BalanceTaskPolicy()
	dstTarget := getBindCcl(tids[0], cpuNumPerCluster, t)
	if dstTarget != srcTarget {
		t.Errorf("migration occurs with gap %d\n", noMigrateGap)
	}

	// test load beyond migration gap
	addCclLoad(srcTarget, cpuNumPerCluster, cpuLoadStub, migrateGap)
	time.Sleep(time.Duration(testSleepTime) * time.Second)
	BalanceTaskPolicy()
	dstTarget = getBindCcl(tids[0], cpuNumPerCluster, t)
	if dstTarget == srcTarget {
		t.Errorf("not migrate with gap %d\n", migrateGap)
	}
	UnbindTaskPolicy(tids[0])
	SwitchCclAware(false)
	testlib.CleanStub()
}

// TestMigrationWithTaskLoad test migration with taskload
func TestMigrationWithTaskLoad(t *testing.T) {
	var block ControlBlock
	topoStub, _, taskStub := testlib.InitStub()
	SwitchCclAware(true)
	if err := Init(); err != nil {
		t.Error("init policy failed")
	}
	tids := taskStub.CreateTasks(groupThreadNum)
	cpuNumPerCluster := topoStub.CPUNum / topoStub.ClusterNum

	// bind a init ccl
	BindTasksPolicy(&block)
	srcTarget := getBindCcl(tids[0], cpuNumPerCluster, t)

	// add task load
	for _, tid := range tids {
		taskStub.AddLoad(tid, migrateGap, testSleepTime)
	}
	time.Sleep(time.Duration(testSleepTime) * time.Second)
	BalanceTaskPolicy()
	for _, tid := range tids {
		dstTarget := getBindCcl(tid, cpuNumPerCluster, t)
		if dstTarget != srcTarget {
			t.Errorf("task laod cause migrate")
		}
	}

	for _, tid := range tids {
		UnbindTaskPolicy(tid)
	}
	SwitchCclAware(false)
	testlib.CleanStub()
}

// TestSwitchCclAware test cclAware switch
func TestSwitchCclAware(t *testing.T) {
	var block ControlBlock
	topoStub, _, taskStub := testlib.InitStub()
	tids := taskStub.CreateTasks(testThreadNum)
	SwitchCoarseGrain(true)
	if err := Init(); err != nil {
		t.Errorf("init policy failed\n")
	}

	// turn ccl aware on
	SwitchCclAware(true)
	cpuNumPerCluster := topoStub.CPUNum / topoStub.ClusterNum
	BindTasksPolicy(&block)
	cpus, err := testlib.GetAffinityStub(tids[0])
	if err != nil {
		t.Errorf("get affinity failed")
	}
	if cpuNumPerCluster != len(cpus) {
		t.Errorf("ccl aware on: affinity cpu number %d is wrong", len(cpus))
	}
	for i := 0; i < len(cpus)-1; i++ {
		if cpus[i]/cpuNumPerCluster != cpus[i+1]/cpuNumPerCluster {
			t.Errorf("bind cpus are not in one cluster\n")
		}
	}
	UnbindTaskPolicy(tids[0])

	// turn ccl aware off
	SwitchCclAware(false)
	cpuNumPerNuma := topoStub.CPUNum / topoStub.NumaNum
	BindTasksPolicy(&block)
	cpus, err = testlib.GetAffinityStub(tids[0])
	if err != nil {
		t.Error(err.Error())
	}
	if cpuNumPerNuma != len(cpus) {
		t.Errorf("ccl aware off: affinity cpu number %d is wrong", len(cpus))
	}
	for i := 0; i < len(cpus)-1; i++ {
		if int(cpus[i])/cpuNumPerNuma != int(cpus[i+1])/cpuNumPerNuma {
			t.Errorf("bind cpus are not in one NUMA\n")
		}
	}
	UnbindTaskPolicy(tids[0])
	testlib.CleanStub()
}

// TestSwitchCoarseGrain test coarseGrain switch
func TestSwitchCoarseGrain(t *testing.T) {
	var block ControlBlock
	_, _, taskStub := testlib.InitStub()
	tids := taskStub.CreateTasks(testThreadNum)
	if err := Init(); err != nil {
		t.Errorf("init policy failed\n")
	}
	// turn coarseGrain on
	SwitchCoarseGrain(true)
	BindTasksPolicy(&block)
	cpus, err := testlib.GetAffinityStub(tids[0])
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Printf("coarseOn cpus %v", cpus)
	if len(cpus) == coarseGrainThreadNum {
		t.Errorf("affinity cpu number is 1 when coarseGrain on")
	}
	UnbindTaskPolicy(tids[0])
	//  turn coarseGrain off
	SwitchCoarseGrain(false)
	BindTasksPolicy(&block)
	cpus, err = testlib.GetAffinityStub(tids[0])
	if err != nil {
		t.Errorf("get affinity failed")
	}
	if len(cpus) != 1 {
		t.Errorf("affinity cpu number is not 1 when coarseGrain off")
	}
	UnbindTaskPolicy(tids[0])
	testlib.CleanStub()
}

// TestBindNumerousTasks test time to bind numerous tasks
func TestBindNumerousTasks(t *testing.T) {
	var block ControlBlock
	_, _, taskStub := testlib.InitStub()
	tasks := make([][]uint64, numerousTaskNum, numerousTaskNum)
	for i := 0; i < numerousTaskNum; i++ {
		tasks[i] = taskStub.CreateTasks(groupThreadNum)
	}
	if err := Init(); err != nil {
		t.Errorf("init policy failed\n")
	}
	BindTasksPolicy(&block)
	time.Sleep(time.Duration(testSleepTime) * time.Second)
	for i := 0; i < numerousTaskNum; i++ {
		for _, tid := range tasks[i] {
			if _, err := testlib.GetAffinityStub(tid); err != nil {
				t.Errorf("bind task failed\n")
			}
			UnbindTaskPolicy(tid)
		}
	}
	testlib.CleanStub()
}
