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

// Package sysload implements utility go get cpu and task load
// from stat files in procfs
package sysload

import (
	"bufio"
	"container/list"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gitee.com/wisdom-advisor/common/utils"
	log "github.com/sirupsen/logrus"
)

// #include <unistd.h>
import "C"

var clockTicks uint64

// ScaleShift is in order to avoid float compute
const ScaleShift uint64 = 10
const nsecPerSec uint64 = 1000000000
const cpuNameIndex = 0
const cpuIDIndex = 3

type timeUsage struct {
	lastUpdate time.Time
	user       uint64
	system     uint64
	load       int
}

func init() {
	clockTicks = uint64(C.sysconf(C._SC_CLK_TCK))
}

func scaleUp(i uint64) uint64 {
	return i << ScaleShift
}

func (usage *timeUsage) init(now *time.Time, user uint64, system uint64) {
	usage.lastUpdate = *now
	usage.user = user
	usage.system = system
}

func (usage *timeUsage) update(now *time.Time, user uint64, system uint64) {
	duration := uint64(now.Sub(usage.lastUpdate)) * clockTicks / nsecPerSec
	if duration == 0 {
		return
	}

	usage.load = int(scaleUp((user-usage.user)+(system-usage.system)) / (duration))
	usage.lastUpdate = *now
	usage.user = user
	usage.system = system
}

// Load represent cpu load or task load with cpu usage
type Load struct {
	id    uint64
	usage timeUsage
}

func (load *Load) set(id uint64) {
	load.id = id
}

// Init of Load init load with timestamp
func (load *Load) Init(now *time.Time, user uint64, system uint64) {
	load.usage.init(now, user, system)
}

// Update of Load update load with new timestamp
func (load *Load) Update(now *time.Time, user uint64, system uint64) {
	load.usage.update(now, user, system)
}

// SystemLoad provides cpus and tasks load according running time
type SystemLoad struct {
	cpusLoad  []Load
	tasksLoad list.List
}

type cpuStat struct {
	cpuNum uint64
	user   uint64
	system uint64
}

func parseCPUStatLine(str string) (cpuStat, error) {
	const cpuUserTimeIndex = 1
	const cpuSystemTimeIndex = 3

	var stat cpuStat
	var err error

	statFields := strings.Fields(str)
	cpu := statFields[cpuNameIndex]
	stat.user, err = strconv.ParseUint(statFields[cpuUserTimeIndex], utils.DecimalBase, utils.Uint64Bits)
	if err != nil {
		err = fmt.Errorf("convert cpu user time from %s failed", statFields[cpuUserTimeIndex])
		return stat, err
	}
	stat.system, err = strconv.ParseUint(statFields[cpuSystemTimeIndex], utils.DecimalBase, utils.Uint64Bits)
	if err != nil {
		err = fmt.Errorf("convert cpu system time from %s failed", statFields[cpuSystemTimeIndex])
		return stat, err
	}
	stat.cpuNum, err = strconv.ParseUint(cpu[cpuIDIndex:], utils.DecimalBase, utils.Uint64Bits)
	if err != nil {
		err = fmt.Errorf("convert cpu number from %s failed", cpu)
		return stat, err
	}

	return stat, nil
}

func (sysload *SystemLoad) cpusLoadUpdate(init bool) {
	file, err := os.Open(utils.ProcDir + "stat")
	if err != nil {
		log.Error(err)
		return
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	now := time.Now()
	for scanner.Scan() {
		line := scanner.Text()
		if line[0:cpuIDIndex] != "cpu" {
			break
		}
		if line[cpuIDIndex] == ' ' {
			continue
		}
		stat, err := parseCPUStatLine(line)
		if err != nil {
			log.Error(err)
			continue
		}
		if init {
			sysload.cpusLoad[stat.cpuNum].Init(&now, stat.user, stat.system)
			sysload.cpusLoad[stat.cpuNum].set(stat.cpuNum)
		} else {
			sysload.cpusLoad[stat.cpuNum].Update(&now, stat.user, stat.system)
		}
	}
	file.Close()
}

type taskStat struct {
	user   uint64
	system uint64
}

func parseTaskStatLine(str string) (taskStat, error) {
	const taskUserTimeIndex = 13
	const taskSystemTimeIndex = 14
	var stat taskStat
	var err error

	statFields := strings.Fields(str)
	stat.user, err = strconv.ParseUint(statFields[taskUserTimeIndex], utils.DecimalBase, utils.Uint64Bits)
	if err != nil {
		err = fmt.Errorf("convert task user time from %s failed", statFields[taskUserTimeIndex])
		return stat, err
	}
	stat.system, err = strconv.ParseUint(statFields[taskSystemTimeIndex], utils.DecimalBase, utils.Uint64Bits)
	if err != nil {
		err = fmt.Errorf("convert task system time from %s failed", statFields[taskSystemTimeIndex])
		return stat, err
	}
	return stat, nil
}

func getPidFromTid(tid uint64) (uint64, error) {
	var pid uint64

	taskStatusPath := fmt.Sprintf(utils.ProcDir+"%d/status", tid)
	file, err := os.Open(taskStatusPath)
	if err != nil {
		return pid, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Tgid") {
			_, err := fmt.Sscanf(line, "Tgid: %d", &pid)
			return pid, err
		}
	}
	return pid, fmt.Errorf("can not find Tgid from status file")
}

func updateTaskLoad(load *Load, init bool) {
	pid, err := getPidFromTid(load.id)
	if err != nil {
		log.Warn(err)
		return
	}

	taskStatPath := fmt.Sprintf(utils.ProcDir+"%d/task/%d/stat", pid, load.id)
	line := utils.ReadAllFile(taskStatPath)
	if line == "" {
		return
	}
	stat, err := parseTaskStatLine(line)
	if err != nil {
		log.Error(err)
		return
	}

	now := time.Now()
	if init {
		load.Init(&now, stat.user, stat.system)
	} else {
		load.Update(&now, stat.user, stat.system)
	}
}

func (sysload *SystemLoad) tasksLoadUpdate() {
	for task := sysload.tasksLoad.Front(); task != nil; task = task.Next() {
		updateTaskLoad((task.Value).(*Load), false)
	}
}

// Init of SystemLoad init cpus' and tasks' load with current timestamp
func (sysload *SystemLoad) Init() {
	sysload.cpusLoad = make([]Load, utils.CPUNum)
	sysload.tasksLoad.Init()
	sysload.cpusLoadUpdate(true)
}

// AddTask of SystemLoad monitor new task's load
func (sysload *SystemLoad) AddTask(id uint64) {
	var taskLoad Load

	taskLoad.id = id
	updateTaskLoad(&taskLoad, true)
	sysload.tasksLoad.PushBack(&taskLoad)
}

// RemoveTask of SystemLoad stop to monitor new task's load
func (sysload *SystemLoad) RemoveTask(id uint64) {
	taskLoad := sysload.findTask(id)
	if taskLoad != nil {
		sysload.tasksLoad.Remove(taskLoad)
	}
}

// Update of SystemLoad update load if cpus and monitored tasks
func (sysload *SystemLoad) Update() {
	sysload.cpusLoadUpdate(false)
	sysload.tasksLoadUpdate()
}

func (sysload *SystemLoad) findTask(id uint64) *list.Element {
	for taskLoad := sysload.tasksLoad.Front(); taskLoad != nil; taskLoad = taskLoad.Next() {
		load, ok := (taskLoad.Value).(*Load)
		if !ok {
			continue
		}
		if load.id == id {
			return taskLoad
		}
	}
	return nil
}

// GetCPULoad of SystemLoad get cpu load
func (sysload *SystemLoad) GetCPULoad(cpu int) int {
	if cpu > len(sysload.cpusLoad) {
		return 0
	}

	return sysload.cpusLoad[cpu].usage.load
}

// GetTaskLoad of SystemLoad get task load
func (sysload *SystemLoad) GetTaskLoad(id uint64) int {
	taskLoad := sysload.findTask(id)
	if taskLoad != nil {
		return (taskLoad.Value).(*Load).usage.load
	}
	return 0
}

// NewSysload alloc a new sysload object
func NewSysload() *SystemLoad {
	var sysload SystemLoad

	sysload.Init()
	return &sysload
}
