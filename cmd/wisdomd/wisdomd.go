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
	"gitee.com/wisdom-advisor/common/policy"
	"gitee.com/wisdom-advisor/common/ptrace"
	"gitee.com/wisdom-advisor/common/utils"
	"log/syslog"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	pb "gitee.com/wisdom-advisor/api/profile"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	wisdomdUsage = `wisdomd daemon is the perf tune daemon
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

var block policy.ControlBlock

type WisdomdServer struct{}

func (s *WisdomdServer) StartUserSetBind(ctx context.Context, in *pb.UserSetPolicy) (*pb.Ack, error) {
	log.Debugf("StartUserSetBind %v", in)

	policy.SwitchAffinityAware(false)
	policy.SwitchNetAware(in.Bind.NetAwareBind)
	policy.SwitchCclAware(in.Bind.CCLBind)
	policy.SwitchPerCore(in.Bind.PerCoreBind)
	policy.BalanceTaskPolicy()
	policy.BindTasksPolicy(&block)

	return &pb.Ack{Status: "OK"}, nil
}

func (s *WisdomdServer) StartAutoThreadAffinityBind(ctx context.Context, in *pb.DetectPolicy) (*pb.Ack, error) {
	log.Debugf("StartAutoThreadAffinityBind %v", in)
	if policy.ShouldStartPtraceScan(&block) {
		return &pb.Ack{Status: "The scan is Running now"}, nil
	}

	if in.Trace.Period > maxPeriod {
		log.Errorf("period invalid, should greater than zero and less than %d", maxPeriod)
		return &pb.Ack{Status: "Period invalid,can not be greater than maxPeriod"}, nil
	}

	policy.SwitchAffinityAware(true)
	policy.SetAffinityTaskName(in.TaskName)
	timer := time.NewTicker(time.Duration(in.Trace.Period) * time.Second)
	policy.SetAffinityTraceTime(int(in.Trace.TraceTime))
	policy.SwitchNetAware(in.Bind.NetAwareBind)
	policy.SwitchCclAware(in.Bind.CCLBind)
	policy.SwitchPerCore(in.Bind.PerCoreBind)

	policy.SwitchPtraceScan(&block, true)

	go func() {
		for {
			select {
			case <-timer.C:
				if !policy.ShouldStartPtraceScan(&block) {
					log.Debugf("StopAutoThreadAffinityBind %v", in)
					return
				}

				policy.BalanceTaskPolicy()
				policy.BindTasksPolicy(&block)
			}
		}
	}()

	return &pb.Ack{Status: "OK"}, nil
}

func (s *WisdomdServer) StartThreadGrouping(ctx context.Context, in *pb.CPUPartition) (*pb.Ack, error) {
	log.Debugf("StartThreadGrouping %v", in)
	var party policy.CPUPartition
	var err error

	if policy.ShouldStartPtraceScan(&block) {
		return &pb.Ack{Status: "The scan is Running now"}, nil
	}
	if in.Trace.Period > maxPeriod {
		log.Errorf("period invalid, should greater than zero and less than %d", maxPeriod)
		return &pb.Ack{Status: "Period invalid,can not be greater than maxPeriod"}, nil
	}

	timer := time.NewTicker(time.Duration(in.Trace.Period) * time.Second)

	policy.SwitchPtraceScan(&block, true)

	if in.IOCPUlist != "" && in.NetCPUlist != "" {
		party, err = policy.ParsePartition(in.IOCPUlist, in.NetCPUlist)
		if err != nil {
			return &pb.Ack{Status: "IO list or net List invalid"}, nil
		}
	} else {
		party = policy.GenerateDefaultPartitions()
	}

	go func() {
		for {
			select {
			case <-timer.C:
				if !policy.ShouldStartPtraceScan(&block) {
					log.Debugf("StopThreadGrouping %v", in)
					return
				}
				if pid, err := utils.GetPid(in.TaskName); err == nil {
					log.Debugf("StartThreadGrouping  pid %d", pid)
					threads, err := ptrace.DoCollect(pid, int(in.Trace.TraceTime), ptrace.ParseSyscall)
					if err != nil {
						log.Error(err)
					}
					policy.BindPartition(party, threads, policy.IONetBindPolicy)
				}
			}
		}
	}()

	return &pb.Ack{Status: "OK"}, nil
}

func (s *WisdomdServer) SetScan(ctx context.Context, in *pb.Switch) (*pb.Ack, error) {
	log.Debugf("SetScan %v", in)
	policy.SwitchPtraceScan(&block, in.Start)
	return &pb.Ack{Status: "OK"}, nil
}

func setLogLevel(levelstring string) {
	level, _ := log.ParseLevel(levelstring)
	log.SetLevel(level)
	return
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
	err := policy.Init()
	return err
}

func runWisdomd(ctx *cli.Context) error {
	listener, err := listenUnixSock(cmdSocketPath)
	if err != nil {
		return err
	}
	interrupt := relayQuitSig()

	grpcServer := grpc.NewServer()
	pb.RegisterWisdomMgrServer(grpcServer, &WisdomdServer{})
	log.Debugf("Wisdomd grpc server start\n")
	go grpcServer.Serve(listener)

	select {
	case <-interrupt:
		goto out
	}
out:
	listener.Close()
	log.Info("quit")
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "wisdomd"
	app.Usage = wisdomdUsage

	app.Version = version
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "printlog",
			Usage: "output log to terminal for debugging",
		},
		cli.StringFlag{
			Name:  "loglevel",
			Value: "info",
			Usage: "log level",
		},
	}

	app.Before = doBeforeJob
	app.Action = runWisdomd

	if err := app.Run(os.Args); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
