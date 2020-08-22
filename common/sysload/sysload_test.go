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

package sysload

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"gitee.com/wisdom-advisor/common/testlib"
	"gitee.com/wisdom-advisor/common/utils"
)

const userHz = 100
const tickLen = nsecPerSec / userHz
const tickLimit = int64((1<<64 - 1) / tickLen)
const constZero = 0
const defCPU = 1
const testCount = 10
const pidLimit = 536870912
const testCountGet = 10
const testSleepTime = 1
const percentLimit = 100
const acceptMarginPct = 3
const testLoadTaskNum = 1

// TestParseCPUStatLine test the parsing of CPU stat lines
func TestParseCPUStatLine(t *testing.T) {
	cpuNum := uint64(defCPU)
	var testStat testlib.CPUProcStat

	testStat.Init(cpuNum)
	stat, err := parseCPUStatLine(testStat.ToString())
	if err != nil {
		t.Errorf("parse line error which is not expected")
	}

	if stat.user != testStat.User {
		t.Errorf(" user %d; expected %d", stat.user, testStat.User)
	}

	if stat.system != testStat.System {
		t.Errorf(" system %d; expected %d", stat.system, testStat.System)
	}

	if stat.cpuNum != testStat.CPUNum {
		t.Errorf(" cpuNum %d; expected %d", stat.cpuNum, testStat.CPUNum)
	}
}

// TestMultiParseStat is multiple test for TestParseCPUStatLine
func TestMultiParseStat(t *testing.T) {
	for i := 0; i < testCount; i++ {
		t.Run("TestParseCPUStatLine", TestParseCPUStatLine)
	}
}

// TestGetPidFromTid test get pid from tid
func TestGetPidFromTid(t *testing.T) {
	utils.ProcDir = "./tmp/proc/"
	var tid = uint64(rand.Int63n(pidLimit))
	var pid = uint64(rand.Int63n(pidLimit))
	path := fmt.Sprintf("./tmp/proc/%d/", tid)

	rand.Seed(time.Now().Unix())
	data := []byte("Name:	sedispatch\n")
	data = append(data, []byte("Umask:	0022\n")...)
	data = append(data, []byte("State:	S (sleeping)\n")...)
	data = append(data, []byte(fmt.Sprintf("Tgid:	%d\n", pid))...)
	data = append(data, []byte("Ngid:	0\n")...)
	data = append(data, []byte(fmt.Sprintf("Pid:	%d\n", tid))...)
	data = append(data, []byte("PPid:	1999\n")...)
	data = append(data, []byte("TracerPid:	0\n")...)
	data = append(data, []byte("Uid:	0	0	0	0\n")...)
	data = append(data, []byte("Gid:	0	0	0	0\n")...)
	data = append(data, []byte("FDSize:	64\n")...)
	data = append(data, []byte("Groups:	 \n")...)
	data = append(data, []byte("NStgid:	2002\n")...)
	data = append(data, []byte("NSpid:	2002\n")...)
	data = append(data, []byte("NSpgid:	1999\n")...)
	data = append(data, []byte("NSsid:	1999\n")...)
	data = append(data, []byte("VmPeak:	    8028 kB\n")...)
	data = append(data, []byte("VmSize:	    8028 kB\n")...)
	data = append(data, []byte("VmLck:	       0 kB\n")...)
	data = append(data, []byte("VmPin:	       0 kB\n")...)
	data = append(data, []byte("VmHWM:	    3192 kB\n")...)
	data = append(data, []byte("VmRSS:	    3044 kB\n")...)
	data = append(data, []byte("RssAnon:	     404 kB\n")...)
	data = append(data, []byte("RssFile:	    2640 kB\n")...)
	data = append(data, []byte("RssShmem:	       0 kB\n")...)
	data = append(data, []byte("VmData:	     380 kB\n")...)
	data = append(data, []byte("VmStk:	     132 kB\n")...)
	data = append(data, []byte("VmExe:	      12 kB\n")...)
	data = append(data, []byte("VmLib:	    5480 kB\n")...)
	data = append(data, []byte("VmPTE:	      52 kB\n")...)
	data = append(data, []byte("VmSwap:	       0 kB\n")...)
	data = append(data, []byte("HugetlbPages:	       0 kB\n")...)
	data = append(data, []byte("CoreDumping:	0\n")...)
	data = append(data, []byte("Threads:	1\n")...)
	data = append(data, []byte("SigQ:	1/13293\n")...)
	data = append(data, []byte("SigPnd:	0000000000000000\n")...)
	data = append(data, []byte("ShdPnd:	0000000000000000\n")...)
	data = append(data, []byte("SigBlk:	0000000000000000\n")...)
	data = append(data, []byte("SigIgn:	fffffffe7ffabefe\n")...)
	data = append(data, []byte("SigCgt:	0000000180004001\n")...)
	data = append(data, []byte("CapInh:	0000000000000000\n")...)
	data = append(data, []byte("CapPrm:	0000000000000000\n")...)
	data = append(data, []byte("CapEff:	0000000000000000\n")...)
	data = append(data, []byte("CapBnd:	0000000000000000\n")...)
	data = append(data, []byte("CapAmb:	0000000000000000\n")...)
	data = append(data, []byte("NoNewPrivs:	0\n")...)
	data = append(data, []byte("Seccomp:	2\n")...)
	data = append(data, []byte("Speculation_Store_Bypass:	unknown\n")...)
	data = append(data, []byte("Cpus_allowed:	f\n")...)
	data = append(data, []byte("Cpus_allowed_list:	0-3\n")...)
	data = append(data, []byte("Mems_allowed:	01\n")...)
	data = append(data, []byte("Mems_allowed_list:	0\n")...)
	data = append(data, []byte("voluntary_ctxt_switches:	27663\n")...)
	data = append(data, []byte("nonvoluntary_ctxt_switches:	3\n")...)
	testlib.BuildFakePathWithData(path, "status", data)

	res, err := getPidFromTid(tid)
	if err != nil {
		t.Errorf("get pid error which is not expected")
	}
	if res != pid {
		t.Errorf(" pid %d; expected %d", res, pid)
	}

	errDel := os.RemoveAll("./tmp/")
	if errDel != nil {
		fmt.Println("Remove path error")
	}
}

// TestMultiGetPid is multiple test for getPidFromTid
func TestMultiGetPid(t *testing.T) {
	for i := 0; i < testCountGet; i++ {
		t.Run("TestGetPidFromTid", TestGetPidFromTid)
	}
}

func checkLoadValid(load int, testLoad int, t *testing.T) {
	var diff int
	scaleTestLoad := int(scaleUp(uint64(testLoad)) / testlib.ConvertPct)
	if scaleTestLoad > load {
		diff = scaleTestLoad - load
	} else {
		diff = load - scaleTestLoad
	}
	if diff > int(scaleUp(acceptMarginPct)/testlib.ConvertPct) {
		t.Errorf("get load %d, scaletestLoad %d, excced accept margin %d\n", load, scaleTestLoad, acceptMarginPct)
	}
}

// TestGetCPULoad test GetCPULoad
func TestGetCPULoad(t *testing.T) {
	var sysload SystemLoad
	topoStub, cpuLoadStub, _ := testlib.InitStub()

	sysload.Init()

	time.Sleep(time.Duration(testSleepTime) * time.Second)

	testLoad := rand.Intn(percentLimit)
	cpuLoadStub.AddLoad(topoStub.CPUNum-1, testLoad, testSleepTime)
	cpuLoadStub.DeployLoad()
	sysload.Update()
	load := sysload.GetCPULoad(topoStub.CPUNum - 1)

	checkLoadValid(load, testLoad, t)
	testlib.CleanStub()
}

// TestGetTaskLoad test GetTaskLoad
func TestGetTaskLoad(t *testing.T) {
	var sysload SystemLoad
	_, _, taskStub := testlib.InitStub()

	tids := taskStub.CreateTasks(testLoadTaskNum)
	sysload.Init()
	sysload.AddTask(tids[0])

	time.Sleep(time.Duration(testSleepTime) * time.Second)

	testLoad := rand.Intn(percentLimit)
	taskStub.AddLoad(tids[0], int(testLoad), testSleepTime)
	sysload.Update()
	load := sysload.GetTaskLoad(tids[0])

	checkLoadValid(load, testLoad, t)
	testlib.CleanStub()
}
