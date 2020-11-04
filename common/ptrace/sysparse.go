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

package ptrace

import (
	"errors"
	"fmt"
	"gitee.com/wisdom-advisor/common/netaffinity"
	"gitee.com/wisdom-advisor/common/utils"
	log "github.com/sirupsen/logrus"
	unix "golang.org/x/sys/unix"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type syscallCount struct {
	NetAccess   uint64
	FutexMap    map[uint64]int
	IOGetEvents uint64
}

// ProcessFeature is to decribe the feature of one process
type ProcessFeature struct {
	Pid      uint64
	SysCount syscallCount
}

// ProcDir is path of proc sysfs
var ProcDir = "/proc/"


func isSocketfd(link string) (bool, uint64) {
	reg := regexp.MustCompile(`socket:\[(\d+)\]`)
	params := reg.FindStringSubmatch(link)
	if params != nil {
		inode, err := strconv.ParseUint(params[1], utils.DecimalBase, utils.Uint64Bits)
		if err == nil {
			return true, inode
		}
	}
	return false, 0
}

func isNetSocket(inode uint64) (bool, net.IP) {
	netFiles := []string{ProcDir + "/net/tcp", ProcDir + "/net/udp", ProcDir + "/net/tcp6", ProcDir + "/net/udp6"}

	for _, file := range netFiles {
		inodes, err := netaffinity.GetInodeIP(file)
		if err != nil {
			log.Error(err)
			continue
		}
		if ip, ok := inodes[inode]; ok {
			return true, ip
		}
	}

	return false, nil
}



func collectSockAccess(process *ProcessFeature, fd uint64) error {
	link, err := os.Readlink(fmt.Sprintf("%s/%d/fd/%d", ProcDir, process.Pid, fd))
	if err != nil {
		return errors.New("collectRead Readlink fail")
	}
	if res, inode := isSocketfd(link); res {
		if res, _ := isNetSocket(inode); res {
			process.SysCount.NetAccess = process.SysCount.NetAccess + 1
		}
	}
	return nil
}

// ParseLoop is the loop which collects syscallinfo
func ParseLoop(pid uint64, stopCh chan int, process *ProcessFeature, wg *sync.WaitGroup, tgid uint64,
	ParseSyscallHandler func(regs unix.PtraceRegs, process *ProcessFeature) error) {
	var status syscall.WaitStatus
	var isEntry = true

	runtime.LockOSThread()

	defer wg.Done()

	if err := Seize(pid); err != nil {
		log.Debug("Seize fail\n")
		return
	}
	if err := Interrupt(pid); err != nil {
		log.Debug("Interrupt fail\n")
		goto out
	}

	for {
		syscall.Wait4(int(pid), &status, syscall.WALL, nil)
		if status.Exited() {
			return
		} else if status.Stopped() {
			break
		}
	}

	for {
		if err := CatchSyscall(pid); err != nil {
			log.Debug("CatchSyscall fail\n")
			goto out
		}

		for {
			if wpid, _ := syscall.Wait4(int(pid), &status,
				syscall.WNOHANG, nil); wpid > 0 {
				if status.Stopped() {
					break
				} else if status.Exited() {
					return
				}
			}
			select {
			case <-stopCh:
				goto out
			default:
			}
		}
		if isEntry {
			regs := CollectSyscall(pid)
			if err := ParseSyscallHandler(regs, process); err != nil {
				log.Info(err)
			}
			isEntry = false
		} else {
			isEntry = true
		}
	}
out:
	if err := Detach(pid); err != nil {
		Interrupt(pid)
		syscall.Wait4(int(pid), &status, syscall.WALL, nil)
		if err := Detach(pid); err != nil {
			log.Debug("Detach fail\n")
		}
	}
}

// DoCollect is to collect the syscall info of one process during timeout time
func DoCollect(pid uint64, timeout int,
	ParseSyscallHandler func(regs unix.PtraceRegs, process *ProcessFeature) error) ([]*ProcessFeature, error) {
	var threadsInfo []*ProcessFeature
	var wg sync.WaitGroup
	stopCh := make(chan int)

	runtime.LockOSThread()

	files, err := ioutil.ReadDir(ProcDir + fmt.Sprintf("%v", pid) + "/task/")
	if err != nil {
		return threadsInfo, errors.New("get process info fail")
	}

	for _, file := range files {
		if tid, err := strconv.ParseUint(file.Name(), utils.DecimalBase, utils.Uint64Bits); err != nil {
			continue
		} else {
			if utils.IsFileExisted((ProcDir + fmt.Sprintf("%v", pid) + "/task/" + file.Name())) {
				var thread ProcessFeature
				thread.Pid = tid
				thread.SysCount.FutexMap = make(map[uint64]int)
				threadsInfo = append(threadsInfo, &thread)
				wg.Add(1)
				go ParseLoop(tid, stopCh, &thread, &wg, pid, ParseSyscallHandler)
			}
		}
	}

	time.Sleep(time.Duration(timeout) * time.Second)

	close(stopCh)
	wg.Wait()

	return threadsInfo, nil
}
