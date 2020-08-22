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

// Package utils provides some commonly used methods
package utils

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// const for function ParseXxx
const (
	DecimalBase      = 10
	HexBase          = 16
	Uint64Bits       = 64
	cpuNumStartIndex = 4
)

// ProcDir is path of proc sysfs
var ProcDir = "/proc/"

// SysDir is path of sysfs
var SysDir = "/sys/"

// CPUNum is cpu number in system
var CPUNum int

// SysCPUPath is the path of CPU related sysfs
var SysCPUPath = "/sys/devices/system/cpu"

var cpuInfoFilePath = "/proc/cpuinfo"
var processorIdentifier = "processor"

func getPhysicalCPUNumber() int {
	f, err := os.Open(cpuInfoFilePath)
	if err != nil {
		return 0
	}
	defer f.Close()

	cpuNum := 0
	s := bufio.NewScanner(f)
	for s.Scan() {
		if err := s.Err(); err != nil {
			return 0
		}

		fields := strings.Fields(s.Text())
		if len(fields) > 0 {
			if fields[0] == processorIdentifier {
				cpuNum++
			}
		}
	}
	return cpuNum
}

func init() {
	CPUNum = getPhysicalCPUNumber()
}

// ReadAllFile read all file to string.
func ReadAllFile(path string) string {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

// IsFileExisted indicates whether the file exists or not
func IsFileExisted(path string) bool {
	_, err := os.Stat(path)
	if err != nil && !os.IsExist(err) {
		return false
	}
	return true
}

// GetPid get the pid of the process with specified comm
func GetPid(name string) (uint64, error) {
	files, err := ioutil.ReadDir(ProcDir)
	if err != nil {
		return 0, errors.New("cannot get pid")
	}
	for _, file := range files {
		tid, err := strconv.ParseUint(file.Name(), DecimalBase, Uint64Bits)
		if err != nil {
			continue
		}
		data, err := ioutil.ReadFile(ProcDir + file.Name() + "/comm")
		if err != nil {
			continue
		}
		if name == strings.Replace(string(data), "\n", "", -1) {
			return tid, nil
		}
	}
	return 0, errors.New("cannot get pid")
}

// GetCPUNumaID is to get the NUMA id of specified CPU
func GetCPUNumaID(cpu int) int {
	var node = -1
	var err error
	cpuDirPath := fmt.Sprintf("%s/cpu%d/", SysCPUPath, cpu)

	dir, err := ioutil.ReadDir(cpuDirPath)
	if err != nil {
		return -1
	}

	for _, f := range dir {
		if strings.HasPrefix(f.Name(), "node") {
			node, err = strconv.Atoi(f.Name()[cpuNumStartIndex:])
			if err != nil {
				node = -1
			}
			break
		}
	}
	return node
}
