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

// Package procscan implements utility to scan procfs to get tasks group
// from process environment variable
package procscan

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gitee.com/wisdom-advisor/common/utils"
	log "github.com/sirupsen/logrus"
)

const limitEnvSize = 1048576 // 1M bytes

// SchedGroupHandler handler to deal with task groups configured by user in environment variables
type SchedGroupHandler struct {
	BindGroup        func(tids []uint64, groupName string)
	UnbindTask       func(tid uint64)
	ForEachBoundTask func(func(tid uint64))
}

func (handler SchedGroupHandler) isMember(path string, member map[string]int) bool {
	data, err := ioutil.ReadFile(path + "/comm")
	if err != nil {
		log.Error(path + " comm read fail")
		return false
	}
	_, ok := member[strings.Replace(string(data), "\n", "", -1)]
	return ok
}

func (handler SchedGroupHandler) bindMembers(name string, member []string, pid uint64) {
	var tids []uint64
	memberMap := make(map[string]int, len(member))
	groupName := fmt.Sprintf("%v", pid) + "_" + name

	for _, line := range member {
		memberMap[line] = 0
	}

	files, err := ioutil.ReadDir(utils.ProcDir + fmt.Sprintf("%v", pid) + "/task/")
	if err != nil {
		return
	}
	for _, file := range files {
		if tid, err := strconv.ParseUint(file.Name(), utils.DecimalBase, utils.Uint64Bits); err != nil {
			continue
		} else {
			fullPath := utils.ProcDir + fmt.Sprintf("%v", pid) + "/task/" + file.Name()
			if handler.isMember(fullPath, memberMap) {
				tids = append(tids, tid)
			}
		}
	}

	handler.BindGroup(tids, groupName)
}

// ParseSchedGroupInfo handle tasks groups of task with pid
func (handler SchedGroupHandler) ParseSchedGroupInfo(pid uint64) {
	reg := regexp.MustCompile(`__SCHED_GROUP__(\S+)=(\S+)`)
	f, err := os.Open(utils.ProcDir + fmt.Sprintf("%v", pid) + "/environ")
	if err != nil {
		log.Debugf(fmt.Sprintf("%v", pid) + " environ open fail")
		return
	}
	defer f.Close()

	buf := bytes.NewBuffer(make([]byte, 0, limitEnvSize))
	_, err = buf.ReadFrom(f)
	if err != nil {
		log.Debugf(fmt.Sprintf("%v", pid) + " environ read fail")
		return
	}

	lines := strings.Split(buf.String(), "\000")
	for _, line := range lines {
		params := reg.FindStringSubmatch(line)
		if params != nil {
			handler.bindMembers(params[1], strings.FieldsFunc(params[2],
				func(c rune) bool { return c == ',' }), pid)
		}
	}
}

// ScanEachProc foreach pid under procfs with func callback
func ScanEachProc(handler func(pid uint64)) {
	files, err := ioutil.ReadDir(utils.ProcDir)
	if err != nil {
		return
	}
	for _, file := range files {
		if pid, err := strconv.ParseUint(file.Name(), utils.DecimalBase, utils.Uint64Bits); err != nil {
			continue
		} else {
			handler(pid)
		}
	}
}

func isTaskExisted(tid uint64) bool {
	return utils.IsFileExisted(utils.ProcDir + fmt.Sprintf("%v", tid))
}

func (handler SchedGroupHandler) checkBoundTask(tid uint64) {
	if !isTaskExisted(tid) {
		handler.UnbindTask(tid)
	}
}

// ScanBoundTask handle exits tasks
func (handler SchedGroupHandler) ScanBoundTask() {
	handler.ForEachBoundTask(handler.checkBoundTask)
}

// Run scan all tasks under procfs to find and handle all tasks groups
func (handler SchedGroupHandler) Run() {
	ScanEachProc(handler.ParseSchedGroupInfo)
	handler.ScanBoundTask()
}
