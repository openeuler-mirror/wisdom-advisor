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
	"errors"
	"gitee.com/wisdom-advisor/common/utils"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const sockPerm = 0700
const bufLimit = 128

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

func relayQuitSig() chan os.Signal {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)
	return ch
}
