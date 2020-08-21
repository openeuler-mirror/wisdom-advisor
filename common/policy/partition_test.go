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
	"gitee.com/wisdom-advisor/common/ptrace"
	"gitee.com/wisdom-advisor/common/sched"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"
)

const defaultPerm = 0644
const (
	testSysCount  = 2
	pidStep       = 1000
	testGroupsNum = 4
	testIOCPUNum  = 4
	testNetCPUNum = 44
)

// TestBindPartition test BindPartition func
func TestBindPartition(t *testing.T) {
	var part CPUPartition
	var threads []*ptrace.ProcessFeature
	var io []int
	var ioNext []int
	var net []int
	var netNext []int
	sched.SetAffinity = setThreadAffinity

	part.FlagCount = make(map[int]int)

	for i := 0; i < 257; i++ {
		var thread ptrace.ProcessFeature
		thread.Pid = uint64(i)
		thread.SysCount.NetAccess = testSysCount
		threads = append(threads, &thread)
	}
	for i := 0; i < 57; i++ {
		var thread ptrace.ProcessFeature
		thread.Pid = uint64(i + pidStep)
		thread.SysCount.IOGetEvents = testSysCount
		threads = append(threads, &thread)
	}

	for i := 0; i < 3; i++ {
		io = append(io, i)
	}

	for i := 93; i < 96; i++ {
		ioNext = append(ioNext, i)
	}

	for i := 3; i < 48; i++ {
		net = append(net, i)
	}

	for i := 48; i < 93; i++ {
		netNext = append(netNext, i)
	}

	PartitionCreateGroup(&part, io, ioAccess)
	PartitionCreateGroup(&part, ioNext, ioAccess)
	PartitionCreateGroup(&part, net, netAccess)
	PartitionCreateGroup(&part, netNext, netAccess)

	BindPartition(part, threads, IONetBindPolicy)
}

func setThreadAffinity(tid uint64, cpu []int) error {
	return nil
}

// TestParseConfig test ParseConfig
func TestParseConfig(t *testing.T) {
	data := []byte("{\n")
	data = append(data, []byte("	\"io\": [\n")...)
	data = append(data, []byte("		\"0-3\",\n")...)
	data = append(data, []byte("		\"92-95\"\n")...)
	data = append(data, []byte("	],\n")...)
	data = append(data, []byte("	\"net\": [\n")...)
	data = append(data, []byte("		\"48-91\",\n")...)
	data = append(data, []byte("		\"4-47\"\n")...)
	data = append(data, []byte("	]\n")...)
	data = append(data, []byte("}\n")...)

	err := ioutil.WriteFile("./tmp.json", data, defaultPerm)
	if err != nil {
		t.Errorf("Create fake json fail\n")
	}
	defer os.Remove("./tmp.json")

	party, err := ParseConfig("./tmp.json")
	if err != nil {
		t.Errorf("Parse json fail")
	}

	if len(party.Groups) != testGroupsNum {
		t.Errorf("Parse json wrong")
	}

	for _, set := range party.Groups {
		if set.Flag == ioAccess && len(set.CPUs) != testIOCPUNum {
			t.Errorf("Parse json wrong")
		}
		if set.Flag == netAccess && len(set.CPUs) != testNetCPUNum {
			t.Errorf("Parse json wrong")
		}
		fmt.Print(set, "\n")
	}

}

// TestStringToInts test StringToInts func
func TestStringToInts(t *testing.T) {
	rand.Seed(time.Now().Unix())

	tmp := rand.Intn(maxCPUNum-1) + 1
	if ret, err := stringToInts(fmt.Sprintf("0-%d", tmp)); err != nil {
		t.Errorf("parse ints string fail\n")
	} else {
		for _, cpu := range ret {
			if cpu < 0 || cpu > tmp {
				t.Errorf("0-9 parse ints string fail\n")
			}
		}
	}

	tmp = rand.Intn(maxCPUNum)
	if ret, err := stringToInts(fmt.Sprintf("%d", tmp)); err != nil {
		t.Errorf("parse ints string fail\n")
	} else {
		for _, cpu := range ret {
			if cpu != tmp {
				t.Errorf("0-9 parse ints string fail\n")
			}
		}
	}
}

// TestGenerateDefaultPartitions test GenerateDefaultPartitions func
func TestGenerateDefaultPartitions(t *testing.T) {
	party := GenerateDefaultPartitions()
	if len(party.Groups) == 0 {
		t.Errorf("Didn't generate groups\n")
	}
	for _, group := range party.Groups {
		if len(group.CPUs) == 0 {
			t.Errorf("Didn't valid generate groups\n")
		}
		fmt.Printf("%d:", group.Flag)
		fmt.Print(group.CPUs)
	}
}
