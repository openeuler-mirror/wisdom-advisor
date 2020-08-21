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
	"fmt"
	"net"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	wisdomUsage = `wisdom is to control some part of wisdom
To get more info of how to use wisdom:
	# wisdom help
`
)

var version = ""

const cmdSocketPath = "/var/run/wisdom.sock"

func sendCmd(context string, path string) error {
	addr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return err
	}

	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return err
	}

	if _, err := conn.Write([]byte(context)); err != nil {
		return err
	}
	return nil
}

func runWisdom(ctx *cli.Context) error {
	cmd := ctx.String("scan")
	if cmd != "start" && cmd != "stop" {
		return errors.New("invalid command")
	}
	if err := sendCmd(fmt.Sprintf("CMD:scan%s", cmd), cmdSocketPath); err != nil {
		return err
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "wisdom"
	app.Usage = wisdomUsage
	app.Version = version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "scan",
			Value: "",
			Usage: "thread feature scan control",
		},
	}

	app.Action = runWisdom

	if err := app.Run(os.Args); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
