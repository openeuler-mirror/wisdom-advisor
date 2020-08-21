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

package topology

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"gitee.com/wisdom-advisor/common/cpumask"
	"gitee.com/wisdom-advisor/common/testlib"
	"gitee.com/wisdom-advisor/common/utils"
)

const testChipNum = 1

func init() {
	rand.Seed(time.Now().Unix())
}

// TestSelectLighterBindNode is to test SelectLighterBindNode
func TestSelectLighterBindNode(t *testing.T) {
	var mask cpumask.Cpumask

	utils.SysDir = "./tmp/sys/"
	topoStub := testlib.NewTopoStub(testChipNum, "./tmp/sys/")

	utils.CPUNum = topoStub.CPUNum
	if err := InitTopo(); err != nil {
		t.Errorf("InitTopo failed\n")
	}

	testCPUNum := rand.Intn(topoStub.CPUNum)
	mask.Set(testCPUNum)
	testCPUNode := findTypeTopoNode(TopoTypeCPU, &mask)
	testCPUNode.SubBind()
	testCPUNode = tree.root.SelectLighterBindNode(TopoTypeCPU)

	if testCPUNode.id != testCPUNum {
		t.Errorf("expect cpu id %d, result %d\n", testCPUNum, testCPUNode.id)
	}
	err := os.RemoveAll("./tmp/")
	if err != nil {
		fmt.Println("Remove path error")
	}
}
