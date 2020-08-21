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

package procscan

import (
	"fmt"
	"os"
	"testing"

	"gitee.com/wisdom-advisor/common/testlib"
	"gitee.com/wisdom-advisor/common/utils"
)

const constZero = 0
const memberSize = 5

// TestIsMember test the member check function
func TestIsMember(t *testing.T) {
	utils.ProcDir = "./tmp/proc/"
	var handler SchedGroupHandler
	var data []byte
	member := make(map[string]int, memberSize)
	path := fmt.Sprintf("./tmp/proc/23/")
	data = append(data, []byte("a1")...)
	var ret bool

	testlib.BuildFakePathWithData(path, "comm", data)
	ret = handler.isMember("./tmp/proc/23/", member)
	if ret != false {
		t.Errorf("expect false, get ture")
	}

	err := os.RemoveAll("./tmp/")
	if err != nil {
		fmt.Println("Remove path error")
	}
}

// TestIsMember2 test the member check function
func TestIsMember2(t *testing.T) {
	utils.ProcDir = "./tmp/proc/"
	var handler SchedGroupHandler
	var data []byte
	member := make(map[string]int, memberSize)
	path := fmt.Sprintf("./tmp/proc/23/")
	data = append(data, []byte("a1")...)
	var ret bool

	member["a1"] = constZero
	testlib.BuildFakePathWithData(path, "comm", data)
	ret = handler.isMember("./tmp/proc/23/", member)
	if ret != true {
		t.Errorf("expect false, get ture")
	}

	err := os.RemoveAll("./tmp/")
	if err != nil {
		fmt.Println("Remove path error")
	}
}

// TestIsMember3 test the member check function with line break
func TestIsMember3(t *testing.T) {
	utils.ProcDir = "./tmp/proc/"
	var handler SchedGroupHandler
	var data []byte
	member := make(map[string]int, memberSize)
	path := fmt.Sprintf("./tmp/proc/23/")
	data = append(data, []byte("a1\n")...)
	var ret bool

	member["a1"] = constZero
	testlib.BuildFakePathWithData(path, "comm", data)
	ret = handler.isMember("./tmp/proc/23/", member)
	if ret != true {
		t.Errorf("expect false, get ture")
	}

	err := os.RemoveAll("./tmp/")
	if err != nil {
		fmt.Println("Remove path error")
	}
}

// TestIsTaskExisted check file existing check
func TestIsTaskExisted(t *testing.T) {
	utils.ProcDir = "./tmp/proc/"

	path := fmt.Sprintf("./tmp/proc/0/")
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		fmt.Println("Mkdir error")
		return
	}

	ret := isTaskExisted(constZero)
	if ret != true {
		t.Errorf("expect false, get ture")
	}

	err = os.RemoveAll("./tmp/")
	if err != nil {
		fmt.Println("Remove path error")
	}
}
