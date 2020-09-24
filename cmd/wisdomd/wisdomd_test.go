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

package main

import (
	"context"
	pb "gitee.com/wisdom-advisor/api/profile"
	"gitee.com/wisdom-advisor/common/policy"
	"testing"
)

func init() {
	policy.Init()
}
func TestStartUserSetBind(t *testing.T) {
	s := WisdomdServer{}

	req := pb.UserSetPolicy{
		Bind: &pb.BindMethod{
			NetAwareBind: true,
			CCLBind:      true,
			PerCoreBind:  true,
		},
	}

	res, _ := s.StartUserSetBind(context.Background(), &req)

	if res.Status != "OK" {
		t.Errorf("StartUserSetBind Fail %s", res.Status)
	}
	s.SetScan(context.Background(), &pb.Switch{Start: false})
}

func TestStartAutoThreadAffinityBind(t *testing.T) {
	s := WisdomdServer{}

	req := pb.DetectPolicy{TaskName: "sem",
		Trace: &pb.TracePara{TraceTime: 5,
			Period: 10},
		Bind: &pb.BindMethod{NetAwareBind: true,
			CCLBind:     true,
			PerCoreBind: true},
	}

	res, _ := s.StartAutoThreadAffinityBind(context.Background(), &req)

	if res.Status != "OK" {
		t.Errorf("StartUserSetBind Fail %s", res.Status)
	}
	s.SetScan(context.Background(), &pb.Switch{Start: false})
}

func TestStartThreadGrouping(t *testing.T) {
	s := WisdomdServer{}

	req := pb.CPUPartition{
		TaskName:   "net_test",
		IOCPUlist:  "1-2,5-6",
		NetCPUlist: "8-9,11-12",
		Trace: &pb.TracePara{TraceTime: 5,
			Period: 10},
	}
	res, _ := s.StartThreadGrouping(context.Background(), &req)
	if res.Status != "OK" {
		t.Errorf("StartUserSetBind Fail %s", res.Status)
	}
	s.SetScan(context.Background(), &pb.Switch{Start: false})
}
