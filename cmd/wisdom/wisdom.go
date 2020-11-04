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
	"log/syslog"
	"net"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	pb "gitee.com/wisdom-advisor/api/profile"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	wisdomUsage = `wisdom is the client to control wisdomd daemon
To get more info of how to use wisdom:
	# wisdom help
`
)

const (
	defaultPeriod    = 10
	defaultTraceTime = 5
	maxPeriod        = 3600
)

var version = ""

const cmdSocketPath = "/var/run/wisdom.sock"

func UnixConnect(addr string, t time.Duration) (net.Conn, error) {
	unix_addr, err := net.ResolveUnixAddr("unix", cmdSocketPath)
	conn, err := net.DialUnix("unix", nil, unix_addr)
	return conn, err
}
func startUserSetBind(ctx *cli.Context) error {
	rpcClient := newClient()
	req := pb.UserSetPolicy{
		Bind: &pb.BindMethod{
			NetAwareBind: ctx.Bool("netaware"),
			CCLBind:      ctx.Bool("cclaware"),
			PerCoreBind:  ctx.Bool("percore"),
		},
	}
	res, err := rpcClient.StartUserSetBind(context.Background(), &req)
	if err != nil {
		log.Fatal("could not greet: ", err)
		return err
	}
	if res.Status != "OK" {
		log.Fatal(res.Status)
	}
	return err
}
func startThreadAffinity(ctx *cli.Context) error {
	rpcClient := newClient()
	req := pb.DetectPolicy{TaskName: ctx.String("task"),
		Trace: &pb.TracePara{TraceTime: ctx.Uint64("tracetime"),
			Period: uint32(ctx.Uint("period"))},
		Bind: &pb.BindMethod{NetAwareBind: ctx.Bool("netaware"),
			CCLBind:     ctx.Bool("cclaware"),
			PerCoreBind: ctx.Bool("percore")},
	}
	res, err := rpcClient.StartAutoThreadAffinityBind(context.Background(), &req)
	if err != nil {
		log.Fatal("could not greet: ", err)
		return err
	}
	if res.Status != "OK" {
		log.Error(res.Status)
	}
	return err
}

func startThreadGrouping(ctx *cli.Context) error {

	rpcClient := newClient()
	req := pb.CPUPartition{
		TaskName:   ctx.String("task"),
		IOCPUlist:  ctx.String("IO"),
		NetCPUlist: ctx.String("net"),
		Trace: &pb.TracePara{TraceTime: ctx.Uint64("tracetime"),
			Period: uint32(ctx.Uint("period"))},
	}
	res, err := rpcClient.StartThreadGrouping(context.Background(), &req)
	if err != nil {
		log.Fatal("could not greet: ", err)
		return err
	}
	if res.Status != "OK" {
		log.Fatal(res.Status)
	}

	return err
}

func SetScan(ctx *cli.Context) error {
	rpcClient := newClient()
	req := pb.Switch{Start: ctx.Command.HasName("start")}
	res, err := rpcClient.SetScan(context.Background(), &req)
	if err != nil {
		log.Fatal("could not greet: ", err)
		return err
	}
	if res.Status != "OK" {
		log.Fatal(res.Status)
	}
	return err
}

func setLogLevel(ctx *cli.Context) {
	levelString := ctx.String("loglevel")

	level, _ := log.ParseLevel(levelString)

	log.SetLevel(level)

	return
}

func setPrintlog(ctx *cli.Context) {
	if !ctx.Bool("printlog") {
		des, err := syslog.New(syslog.LOG_NOTICE, "Wisdomd")
		if err == nil {
			log.SetOutput(des)
		}
	}
	return
}

func newClient() pb.WisdomMgrClient {
	conn, err := grpc.Dial(cmdSocketPath, grpc.WithInsecure(), grpc.WithDialer(UnixConnect))
	if err != nil {
		log.Fatal("did not connect: ", err)
		return nil
	}
	rpcClient := pb.NewWisdomMgrClient(conn)
	return rpcClient
}

func main() {
	app := cli.NewApp()
	app.Name = "wisdom"
	app.Usage = wisdomUsage
	app.Version = version
	app.EnableBashCompletion = true

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "loglevel",
			Value: "info",
			Usage: "log level",
		},
		&cli.BoolFlag{
			Name:  "printlog",
			Usage: "output log to terminal for debugging",
		},
	}
	app.Before = func(ctx *cli.Context) error {
		setPrintlog(ctx)
		setLogLevel(ctx)
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:   "threadsaffinity",
			Usage:  "trace syscall futex to get thread affinity",
			Action: startThreadAffinity,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "task",
					Usage:    "the name of the task which needs to be affinity aware",
					Required: true,
				},
				&cli.UintFlag{
					Name:  "period",
					Value: defaultPeriod,
					Usage: "scan and balance period",
				},
				&cli.Uint64Flag{
					Name:  "tracetime",
					Value: defaultTraceTime,
					Usage: "time of tracing",
				},
				&cli.BoolFlag{
					Name:  "netaware",
					Usage: "enable net affinity Aware",
				},
				&cli.BoolFlag{
					Name:  "cclaware",
					Usage: "bind thread group inside same cluster",
				},
				&cli.BoolFlag{
					Name:  "percore",
					Usage: "bind one thread per core",
				},
			},
		},

		{
			Name:   "usersetaffinity",
			Usage:  "parse __SCHED_GROUP__ to get thread affinity",
			Action: startUserSetBind,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "netaware",
					Usage: "enable net affinity Aware",
				},
				&cli.BoolFlag{
					Name:  "cclaware",
					Usage: "bind thread group inside same cluster",
				},
				&cli.BoolFlag{
					Name:  "percore",
					Usage: "bind one thread per core",
				},
			},
		},

		{
			Name:   "threadsgrouping",
			Usage:  "trace net and IO syscall, partition threads by user define",
			Action: startThreadGrouping,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "task",
					Usage:    "the name of the task which needs to be threads grouping",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "IO",
					Usage: "partition description for IO,like 0-31,64-95",
				},
				&cli.StringFlag{
					Name:  "net",
					Usage: "partition description for net,like 32-63",
				},
				&cli.UintFlag{
					Name:  "period",
					Value: defaultPeriod,
					Usage: "scan and balance period",
				},
				&cli.Uint64Flag{
					Name:  "tracetime",
					Value: defaultTraceTime,
					Usage: "time of tracing",
				},
			},
		},

		{
			Name:  "scan",
			Usage: "thread feature scan control",
			Subcommands: []cli.Command{
				{
					Name:   "start",
					Action: SetScan,
				},
				{
					Name:   "stop",
					Action: SetScan,
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
