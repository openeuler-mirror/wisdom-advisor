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

package netaffinity

import (
	"fmt"
	"gitee.com/wisdom-advisor/common/utils"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
)

// TestAllGetDevRelated test GetDevRelated func
func TestAllGetDevRelated(t *testing.T) {
	var sshdNum int
	var resNum int

	sshdNum = 0
	resNum = 0

	files, err := ioutil.ReadDir(ProcDir)
	if err != nil {
		return
	}
	for _, file := range files {
		if pid, err := strconv.ParseUint(file.Name(), utils.DecimalBase, utils.Uint64Bits); err != nil {
			continue
		} else {
			devs, err := detectDevRelated(uint64(pid))
			if err != nil {
				fmt.Print(err)
				continue
			}

			if len(devs) == 0 {
				continue
			}
			tmp, err := ioutil.ReadFile(ProcDir + file.Name() + "/cmdline")
			if err != nil {
				fmt.Printf("Get cmdline fail\n")
				continue
			}

			comm, err := ioutil.ReadFile(ProcDir + file.Name() + "/comm")
			if err != nil {
				fmt.Printf("Get cmdline fail\n")
				continue
			}

			if string(comm) == "sshd" {
				sshdNum = sshdNum + 1
			}

			fmt.Printf("%s: %s\n", file.Name(), strings.Replace(string(tmp), "\n", "", -1))
			for _, dev := range devs {
				fmt.Printf("	|%s---node:%d\n", dev.Name, dev.PCINode)
				resNum = resNum + 1
			}
		}
	}
	if sshdNum > 1 && resNum == 0 {
		t.Errorf("bind task early\n")
	}
}
