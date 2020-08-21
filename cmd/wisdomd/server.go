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
	"bytes"
	"errors"
	"gitee.com/wisdom-advisor/common/policy"
	"gitee.com/wisdom-advisor/common/utils"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

const sockPerm = 0700
const bufLimit = 128

func commandHandler(conn net.Conn, cmdMap map[string]func(block *policy.ControlBlock), block *policy.ControlBlock) {
	buf := make([]byte, 0, bufLimit)

	if _, err := conn.Read(buf); err != nil {
		conn.Close()
		return
	}
	if bytes.HasPrefix(buf, []byte("CMD:")) {
		comm := bytes.TrimPrefix(buf, []byte("CMD:"))
		command := strings.Split(string(comm), "\000")
		cmd := command[0]

		if handler, ok := cmdMap[cmd]; ok {
			handler(block)
		}
	}
	conn.Close()
}

func listenUnixSock(path string) (*net.UnixListener, error) {
	if utils.IsFileExisted(path) {
		return nil, errors.New("Sock file " + path + " already exist")
	}

	addr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return nil, errors.New("resolve unix addr fail")
	}

	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, errors.New("create Unix socket fail")
	}

	if err := os.Chmod(path, sockPerm); err != nil {
		return nil, err
	}

	return listener, nil
}

func cmdWaitLoop(listener *net.UnixListener, block *policy.ControlBlock, wg *sync.WaitGroup) {
	defer wg.Done()

	cmdMap := map[string]func(block *policy.ControlBlock){
		"scanstart": policy.PtraceScanStart,
		"scanstop":  policy.PtraceScanEnd,
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Info("listener close")
			break
		}
		commandHandler(conn, cmdMap, block)
	}
}

func relayQuitSig() chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)
	return ch
}
