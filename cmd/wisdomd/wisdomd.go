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

// Package implements a application to set CPU affinity automatic for reduce access cross-NUMA memory
package main

import (
	"errors"
	"log/syslog"
	"os"
	"sync"
	"time"

	"gitee.com/wisdom-advisor/common/policy"
	"gitee.com/wisdom-advisor/common/ptrace"
	"gitee.com/wisdom-advisor/common/utils"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	wisdomdUsage = `wisdomd daemon is the perf tune daemon
Two policies are supported includes threadsaffinity and threadsgrouping.
CPU partition description json script should be provided if threadsgrouping is specified.
The script is like:
{
        "io": [
                "0-4",
                "93-96"
        ],
        "net": [
                "48-92",
                "5-47"
        ]
}
To get more info of how to use wisdomd:
	# wisdomd help
`
)

var version = ""

const cmdSocketPath = "/var/run/wisdom.sock"

const (
	defaultPeriod    = 10
	defaultTraceTime = 5
	maxPeriod        = 3600
)

func setLogLevel(level string) {
	switch level {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warning":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	default:
		log.Infof("invalid level %s, default set to info level", level)
	}
}

func redirectToSyslog() {
	des, e := syslog.New(syslog.LOG_NOTICE, "Wisdomd")
	if e == nil {
		log.SetOutput(des)
	}
}

func doBeforeJob(ctx *cli.Context) error {
	if !ctx.Bool("printlog") {
		redirectToSyslog()
	}
	setLogLevel(ctx.String("loglevel"))

	period := ctx.Int("period")
	if period <= 0 || period > maxPeriod {
		log.Errorf("period invalid, should greater than zero and less than %d", maxPeriod)
		return errors.New("period invalid")
	}

	if ctx.String("policy") == "threadsaffinity" {
		policy.SwitchNumaAware(ctx.Bool("autonuma"))
		policy.SwitchNetAware(ctx.Bool("netaware"))
		policy.SwitchCclAware(ctx.Bool("cclaware"))
		policy.SwitchCoarseGrain(ctx.Bool("coarsegrain"))

		if ctx.Bool("affinityAware") {
			tracetime := ctx.Int("tracetime")
			if tracetime <= 0 || tracetime >= period {
				log.Errorf("tracetime invalid, should greater than zero and less than period %d", period)
				return errors.New("tracetime invalid")
			}
			policy.SetAffinityTraceTime(ctx.Int("tracetime"))
			policy.SetAffinityTaskName(ctx.String("task"))
			policy.SwitchAffinityAware(ctx.Bool("affinityAware"))
		}
	} else if ctx.String("policy") == "threadsgrouping" {
		tracetime := ctx.Int("tracetime")
		if tracetime <= 0 || tracetime >= period {
			log.Errorf("tracetime invalid, should greater than zero and less than period %d", period)
			return errors.New("tracetime invalid")
		}
	} else {
		return errors.New("invalid policy")
	}
	err := policy.Init()
	return err
}

func threadsAffinityLoop(ctx *cli.Context) error {
	var block policy.ControlBlock
	var wg sync.WaitGroup

	timer := time.NewTicker(time.Duration(ctx.Int("period")) * time.Second)
	ch := relayQuitSig()

	if ctx.Bool("affinityAware") {
		policy.PtraceScanStart(&block)
	}

	listener, err := listenUnixSock(cmdSocketPath)
	if err != nil {
		return err
	}

	wg.Add(1)
	go cmdWaitLoop(listener, &block, &wg)

	for {
		select {
		case <-ch:
			goto out
		case <-timer.C:
			policy.BalanceTaskPolicy()
			policy.BindTasksPolicy(&block)
		}
	}
out:
	listener.Close()
	wg.Wait()
	log.Info("quit")
	return nil
}

func threadsGroupingLoop(ctx *cli.Context) error {
	timer := time.NewTicker(time.Duration(ctx.Int("period")) * time.Second)
	ch := relayQuitSig()
	var party policy.CPUPartition
	var block policy.ControlBlock
	var wg sync.WaitGroup

	policy.PtraceScanStart(&block)

	if ctx.String("json") != "" {
		if tmp, err := policy.ParseConfig(ctx.String("json")); err != nil {
			return err
		} else if len(tmp.Groups) == 0 {
			return errors.New("fail to get vaild partition")
		} else {
			party = tmp
		}
	} else {
		log.Info("use default partition")
		party = policy.GenerateDefaultPartitions()
	}

	listener, err := listenUnixSock(cmdSocketPath)
	if err != nil {
		return err
	}

	wg.Add(1)
	go cmdWaitLoop(listener, &block, &wg)

	for {
		select {
		case <-ch:
			goto out
		case <-timer.C:
			if !policy.ShouldStartPtraceScan(&block) {
				continue
			}
			if pid, err := utils.GetPid(ctx.String("task")); err == nil {
				log.Infof("target pid %d", pid)
				threads, err := ptrace.DoCollect(pid, ctx.Int("tracetime"), ptrace.ParseSyscall)
				if err != nil {
					log.Error(err)
				}
				policy.BindPartition(party, threads, policy.IONetBindPolicy)
			}
		}
	}
out:
	listener.Close()
	wg.Wait()
	log.Info("quit")
	return nil
}

func runWisdomd(ctx *cli.Context) error {
	if ctx.String("policy") == "threadsaffinity" {
		return threadsAffinityLoop(ctx)
	} else if ctx.String("policy") == "threadsgrouping" {
		return threadsGroupingLoop(ctx)
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "wisdomd"
	app.Usage = wisdomdUsage

	app.Version = version
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "autonuma",
			Usage: "turn on numa aware schedule",
		},
		cli.BoolFlag{
			Name:  "cclaware",
			Usage: "bind thread group inside same cluster",
		},
		cli.BoolFlag{
			Name:  "coarsegrain",
			Usage: "bind thread in coarse grain",
		},
		cli.BoolFlag{
			Name:  "printlog",
			Usage: "output log to terminal for debugging",
		},
		cli.StringFlag{
			Name:  "loglevel",
			Value: "info",
			Usage: "log level",
		},
		cli.IntFlag{
			Name:  "period",
			Value: defaultPeriod,
			Usage: "scan and balance period",
		},
		cli.BoolFlag{
			Name:  "affinityAware",
			Usage: "enable thread affinity Aware",
		},
		cli.StringFlag{
			Name:  "task",
			Value: "",
			Usage: "the name of the task which needs to be affinity aware",
		},
		cli.Uint64Flag{
			Name:  "tracetime",
			Value: defaultTraceTime,
			Usage: "time of tracing",
		},
		cli.BoolFlag{
			Name:  "netaware",
			Usage: "enable net affinity Aware",
		},
		cli.StringFlag{
			Name:  "policy",
			Value: "threadsaffinity",
			Usage: "specify policy which can be threadsaffinity or threadsgrouping",
		},
		cli.StringFlag{
			Name:  "json",
			Value: "",
			Usage: "CPU partition description script",
		},
	}

	app.Before = doBeforeJob
	app.Action = runWisdomd

	if err := app.Run(os.Args); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
