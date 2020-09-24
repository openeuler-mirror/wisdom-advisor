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

// Package ptrace provides functions for ptrace scanning
package ptrace

import (
	"fmt"
	"unsafe"
	unix "golang.org/x/sys/unix"
	"syscall"
)

// Ptrace wraps ptrace syscall
func Ptrace(request uint64, pid uint64, addr uint64, data uint64) error {
	if _, _, errno := syscall.RawSyscall6(unix.SYS_PTRACE,
		uintptr(request), uintptr(pid), uintptr(addr), uintptr(data), 0, 0); errno != 0 {
		return fmt.Errorf("Ptrace fail no:%d", errno)
	}
	return nil
}

// Seize is to seize one thread which should be done before collecting
func Seize(pid uint64) error {
	return Ptrace(unix.PTRACE_SEIZE, pid, 0, 0)
}

// Detach is to end the seizing of one thread
func Detach(pid uint64) error {
	return Ptrace(unix.PTRACE_DETACH, pid, 0, 0)
}

// Interrupt is to interrupt one thread under seizing
func Interrupt(pid uint64) error {
	return Ptrace(unix.PTRACE_INTERRUPT, pid, 0, 0)
}

// Continue is to continue one thread being interrupted
func Continue(pid uint64) error {
	return Ptrace(unix.PTRACE_CONT, pid, 0, uint64(syscall.SIGTRAP))
}

// CatchSyscall is to interrupt the thread at next syscall
func CatchSyscall(pid uint64) error {
	return Ptrace(unix.PTRACE_SYSCALL, pid, 0, 0)
}

func CollectSyscall(pid uint64) unix.PtraceRegs {
	var regs unix.PtraceRegs
	var iovec syscall.Iovec

	iovec.Base = (*byte)(unsafe.Pointer(&regs))
	iovec.Len = uint64(unsafe.Sizeof(regs))
	Ptrace(syscall.PTRACE_GETREGSET, pid, 1, uint64(uintptr(unsafe.Pointer(&iovec))))

	return regs
}