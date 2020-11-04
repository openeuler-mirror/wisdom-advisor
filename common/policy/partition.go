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
	"errors"
	"gitee.com/wisdom-advisor/common/ptrace"
	"gitee.com/wisdom-advisor/common/sched"
	"gitee.com/wisdom-advisor/common/utils"
	log "github.com/sirupsen/logrus"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const (
	limitEnvSize           = 1048576
	maxCPUNum              = 2048
	AssmuedNodeNum         = 4
	AssmuedSysCallClassNum = 5
)
const (
	ioAccess  = 0
	netAccess = 1
)

// CPUGroup describe one CPU group
type CPUGroup struct {
	Flag int
	CPUs []int
}

// CPUPartition decribe the CPU partition
type CPUPartition struct {
	FlagCount map[int]int
	Groups    []CPUGroup
}

// PartitionCreateGroup is to create one CPU group in the partition
func PartitionCreateGroup(part *CPUPartition, CPUs []int, flag int) {
	var group CPUGroup

	group.CPUs = append(group.CPUs, CPUs...)
	group.Flag = flag

	PartitionAddGroup(part, group)
}

// PartitionAddGroup is to add one group to the partition
func PartitionAddGroup(part *CPUPartition, group CPUGroup) {
	part.Groups = append(part.Groups, group)

	if _, ok := part.FlagCount[group.Flag]; !ok {
		part.FlagCount[group.Flag] = 1
	} else {
		part.FlagCount[group.Flag] = part.FlagCount[group.Flag] + 1
	}
}

// BindPartition is to bind the threads to the partition according to specific policy
func BindPartition(party CPUPartition, threads []*ptrace.ProcessFeature,
	handler func(party CPUPartition, process []*ptrace.ProcessFeature)) {
	handler(party, threads)
}

// IONetBindPolicy is the thread grouping policy according to IO and net access
func IONetBindPolicy(party CPUPartition, threads []*ptrace.ProcessFeature) {
	var netThreads []*ptrace.ProcessFeature
	var IOThreads []*ptrace.ProcessFeature
	var netCPUs []CPUGroup
	var IOCPUs []CPUGroup

	for _, thread := range threads {
		if thread.SysCount.NetAccess > 0 {
			netThreads = append(netThreads, thread)
		}
		if thread.SysCount.IOGetEvents > 0 {
			IOThreads = append(IOThreads, thread)
		}
	}

	for _, group := range party.Groups {
		if group.Flag == ioAccess {
			IOCPUs = append(IOCPUs, group)
		}
		if group.Flag == netAccess {
			netCPUs = append(netCPUs, group)
		}
	}
	bindThreadsToGroups(IOCPUs, IOThreads)
	bindThreadsToGroups(netCPUs, netThreads)
}

func bindThreadsToGroups(CPUset []CPUGroup, threads []*ptrace.ProcessFeature) {
	var base int

	if len(threads) == 0 {
		return
	}

	for _, set := range CPUset {
		base = base + len(set.CPUs)
	}

	if base == 0 {
		return
	}

	for _, set := range CPUset {
		round := (len(threads)*len(set.CPUs) + len(set.CPUs)) / base
		if round >= len(threads) {
			round = len(threads)
			bindThreadsToGroup(set, threads[0:round])
			return
		}
		bindThreadsToGroup(set, threads[0:round])
		threads = threads[round:]
		base = base - len(set.CPUs)
		if base < 0 {
			return
		}
	}
}

func bindThreadsToGroup(CPUset CPUGroup, threads []*ptrace.ProcessFeature) {
	for _, thread := range threads {
		log.Info("bind ", thread.Pid, " to cpu ", CPUset.CPUs)
		if err := sched.SetAffinity(thread.Pid, CPUset.CPUs); err != nil {
			log.Info("bind error")
		}
	}
}

func stringToInts(ints string) ([]int, error) {
	var ret []int

	reg := regexp.MustCompile(`\s*(\d+)\s*-\s*(\d+)\s*`)
	params := reg.FindStringSubmatch(ints)
	if params != nil {
		floor, err := strconv.ParseUint(params[1], utils.DecimalBase, utils.Uint64Bits)
		if err != nil {
			return ret, errors.New("wrong CPU num")
		}
		ceil, err := strconv.ParseUint(params[2], utils.DecimalBase, utils.Uint64Bits)
		if err != nil {
			return ret, errors.New("wrong CPU num")
		}
		if floor > ceil || ceil > maxCPUNum {
			return ret, errors.New("wrong CPU num")
		}

		for i := floor; i <= ceil; i++ {
			ret = append(ret, int(i))
		}
	} else {
		cpu, err := strconv.ParseUint(ints, utils.DecimalBase, utils.Uint64Bits)
		if err != nil {
			return ret, errors.New("wrong CPU num")
		}
		ret = append(ret, int(cpu))
	}
	return ret, nil
}

func ParsePartition(IO string, net string) (CPUPartition, error) {
	var party CPUPartition
	IOlist := strings.Split(IO, ",")
	netList := strings.Split(net, ",")
	party.FlagCount = make(map[int]int, AssmuedSysCallClassNum)

	for _, set := range IOlist {
		cpus, err := stringToInts(set)
		if err != nil {
			return party, err
		}
		log.Debug(cpus)
		PartitionCreateGroup(&party, cpus, ioAccess)
	}
	for _, set := range netList {
		cpus, err := stringToInts(set)
		if err != nil {
			return party, err
		}
		log.Debug(cpus)
		PartitionCreateGroup(&party, cpus, netAccess)
	}
	return party, nil
}

func GenerateDefaultPartitions() CPUPartition {
	var party CPUPartition
	party.FlagCount = make(map[int]int, AssmuedSysCallClassNum)

	numaMap := make(map[int]*CPUGroup, AssmuedNodeNum)

	for i := 0; i < runtime.NumCPU(); i++ {
		numaID := utils.GetCPUNumaID(i)
		if _, ok := numaMap[numaID]; !ok {
			numaMap[numaID] = new(CPUGroup)
			numaMap[numaID].Flag = netAccess
		}
		numaMap[numaID].CPUs = append(numaMap[numaID].CPUs, i)
	}

	for _, group := range numaMap {
		PartitionAddGroup(&party, *group)
	}
	return party
}
