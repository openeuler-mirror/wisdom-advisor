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
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"gitee.com/wisdom-advisor/common/policy"
	"gitee.com/wisdom-advisor/common/testlib"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const testWaitDur = 10
const testThreadNum = 4
const validPeriod = 10
const invalidPeriod1 = 3700
const invalidPeriod2 = -1

// TestFullProcedure is to test the whole procedure
func TestFullProcedure(t *testing.T) {
	_, _, taskStub := testlib.InitStub()
	tids := taskStub.CreateTasks(testThreadNum)

	os.Args = os.Args[:1]
	os.Args = append(os.Args, "--policy")
	os.Args = append(os.Args, "threadsaffinity")
	os.Args = append(os.Args, "--cclaware")
	os.Args = append(os.Args, "--loglevel")
	os.Args = append(os.Args, "info")
	os.Args = append(os.Args, "--period")
	os.Args = append(os.Args, "10")
	os.Args = append(os.Args, "--printlog")
	go main()

	time.Sleep(time.Duration(testWaitDur+1) * time.Second)

	for _, tid := range tids {
		if _, err := testlib.GetAffinityStub(tid); err != nil {
			t.Errorf("tid %d set affinity failed\n", tid)
		}
	}

	for _, tid := range tids {
		policy.UnbindTaskPolicy(tid)
	}
	testlib.CleanStub()
	os.Remove(cmdSocketPath)
}

func createCtxFromFlag(flag *flag.FlagSet) *cli.Context {
	app := cli.NewApp()
	ctx := cli.NewContext(app, flag, nil)
	return ctx
}

// TestSetLogLevel test --loglevel argument
func TestSetLogLevel(t *testing.T) {
	type testLevelData struct {
		setLevel    string
		expectLevel log.Level
	}

	data := []testLevelData{
		{string("invalid"), log.InfoLevel},
		{string("debug"), log.DebugLevel},
		{string("info"), log.InfoLevel},
		{string("warning"), log.WarnLevel},
		{string("error"), log.ErrorLevel},
		{string("fatal"), log.FatalLevel},
		{string("panic"), log.PanicLevel},
	}

	for _, d := range data {
		set := flag.NewFlagSet("", 0)
		set.String("policy", "threadsaffinity", "policy")
		policyData := []string{"--policy", "threadsaffinity"}
		set.String("loglevel", "info", "log level")
		loglevelData := []string{"--loglevel", d.setLevel}
		// shut up error where period is not set
		set.Int("period", defaultPeriod, "scan and balance period")
		periodData := []string{"--period", fmt.Sprintf("%d", validPeriod)}
		data := append(loglevelData, periodData...)
		data = append(data, policyData...)
		set.Parse(data)
		ctx := createCtxFromFlag(set)
		if err := doBeforeJob(ctx); err != nil {
			t.Errorf("set loglevel %s failed\n", d.setLevel)
		}
		if log.GetLevel() != d.expectLevel {
			t.Errorf("test level %s, expect level %d, actual %d\n", d.setLevel, d.expectLevel, log.GetLevel())
		}
	}
}

func testPeriod(p int, t int, policy string) error {
	set := flag.NewFlagSet("", 0)
	set.Int("period", defaultPeriod, "scan and balance period")
	data := []string{"--period", fmt.Sprintf("%d", p)}
	set.String("policy", "threadsaffinity", "policy")
	policyData := []string{"--policy", policy}
	set.Int("tracetime", defaultPeriod, "tracetime")
	traceTime := []string{"--tracetime", fmt.Sprintf("%d", t)}
	set.Bool("affinityAware", false, "affinityAware")
	affinityAware := []string{"--affinityAware"}
	data = append(data, policyData...)
	data = append(data, traceTime...)
	data = append(data, affinityAware...)
	set.Parse(data)
	ctx := createCtxFromFlag(set)
	return doBeforeJob(ctx)
}

func testSetPeriod(t *testing.T, policy string) {
	invalidData := []int{invalidPeriod1, invalidPeriod2, 0}

	for _, p := range invalidData {
		err := testPeriod(p, 1, "threadsaffinity")
		if err == nil {
			t.Errorf("expect error with period %d\n", p)
		}
	}
	if err := testPeriod(validPeriod, 1, "threadsaffinity"); err != nil {
		t.Errorf("set period %d return err %s\n", validPeriod, err.Error())
	}

	for _, p := range invalidData {
		err := testPeriod(p, 1, "threadsaffinity")
		if err == nil {
			t.Errorf("expect error with period %d\n", p)
		}
	}
	if err := testPeriod(validPeriod, 1, "threadsaffinity"); err != nil {
		t.Errorf("set period %d return err %s\n", validPeriod, err.Error())
	}
}

// TestSetPeriod test --period argument
func TestSetPeriod(t *testing.T) {
	testSetPeriod(t, "threadsaffinity")
	testSetPeriod(t, "threadsgrouping")
}

func testSetTraceTime(t *testing.T, policy string) {
	if err := testPeriod(validPeriod, 0, policy); err == nil {
		t.Errorf("expect error with tracetime\n")
	}
	if err := testPeriod(validPeriod, -1, policy); err == nil {
		t.Errorf("expect error with tracetime\n")
	}
	if err := testPeriod(validPeriod, validPeriod+1, policy); err == nil {
		t.Errorf("expect error with tracetime\n")
	}
	if err := testPeriod(validPeriod, 1, policy); err != nil {
		t.Errorf("set period %d return err %s\n", validPeriod, err.Error())
	}
}

// TestSetTraceTime --tracetime argument
func TestSetTraceTime(t *testing.T) {
	testSetTraceTime(t, "threadsaffinity")
	testSetTraceTime(t, "threadsgrouping")
}
