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

package sched

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"gitee.com/wisdom-advisor/common/utils"
)

const targetCPU = 0
const initLen = 1
const cpuFiledIndex = 5

// TestsetAffinity test function setAffinity
func TestSetAffinity(t *testing.T) {
	var cpu int64

	bashCmd := exec.Command("/bin/bash")
	if err := bashCmd.Start(); err != nil {
		t.Errorf("start command bash failed\n")
	}
	pid := bashCmd.Process.Pid

	cpus := make([]int, initLen, initLen)
	cpus[0] = targetCPU
	if err := setAffinity(uint64(pid), cpus); err != nil {
		t.Errorf("setAffinity failed: %s\n", err.Error())
	}
	tasksetCmd := exec.Command("bash", "-c", fmt.Sprintf("/usr/bin/taskset -pc %d", pid))
	out, err := tasksetCmd.Output()
	if err != nil {
		t.Errorf("run taskset failed: %s\n", err.Error())
	}
	fields := strings.Fields(string(out))
	cpu, err = strconv.ParseInt(fields[cpuFiledIndex], utils.DecimalBase, utils.Uint64Bits)
	if err != nil {
		t.Errorf("convert %s to int failed\n", fields[cpuFiledIndex])
	}
	if cpu != targetCPU {
		t.Errorf("expect bind cpu %d, actual %d\n", targetCPU, cpu)
	}
	exec.Command("bash", "-c", fmt.Sprintf("kill -9 %d", pid)).Run()
}
