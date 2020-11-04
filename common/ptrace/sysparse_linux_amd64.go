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
	unix "golang.org/x/sys/unix"
)


// FutextDetect is to Detect Futex
func FutextDetect(regs unix.PtraceRegs, process *ProcessFeature) error {
	if regs.Orig_rax != unix.SYS_FUTEX {
		return nil
	}

	if _, ok := process.SysCount.FutexMap[regs.Rdi]; !ok {
		process.SysCount.FutexMap[regs.Rdi] = 1
	} else {
		process.SysCount.FutexMap[regs.Rdi] = process.SysCount.FutexMap[regs.Rdi] + 1
	}
	return nil
}

// ParseSyscall is to Parse syscall info
func ParseSyscall(regs unix.PtraceRegs, process *ProcessFeature) error {
	sysMap := map[uint64]func(regs unix.PtraceRegs, process *ProcessFeature) error{
		unix.SYS_READ:     CollectSockAccess,
		unix.SYS_WRITE:    CollectSockAccess,
		unix.SYS_SENDTO:   CollectSockAccess,
		unix.SYS_RECVFROM: CollectSockAccess,
		unix.SYS_IO_GETEVENTS: func(regs unix.PtraceRegs, process *ProcessFeature) error {
			process.SysCount.IOGetEvents = process.SysCount.IOGetEvents + 1
			return nil
		},
	}
	if handler, ok := sysMap[regs.Orig_rax]; ok {
		return handler(regs, process)
	}
	return nil
}

// CollectSockAccess is to collect net accesses through fd
func CollectSockAccess(regs unix.PtraceRegs, process *ProcessFeature) error {
	return collectSockAccess(process, regs.Rdi)
}


