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

package netaffinity

import (
	"errors"
	"fmt"
	"gitee.com/wisdom-advisor/common/utils"
	log "github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strconv"
	"syscall"
)

// GetInode can get the inode num of one file
func GetInode(path string) (uint64, error) {
	var stat syscall.Stat_t

	file, err := os.Open(path)
	if err != nil {
		return 0, errors.New("open Inode path fail")
	}
	defer file.Close()

	if err := syscall.Fstat(int(file.Fd()), &stat); err != nil {
		return 0, err
	}

	return stat.Ino, nil
}

// GetPidNSInode can get the inode of the net namespace of specified process
func GetPidNSInode(pid uint64) (uint64, error) {
	reg := regexp.MustCompile(`net:\[(\d+)\]`)

	link, err := os.Readlink(fmt.Sprintf("%s/%d/ns/net", ProcDir, pid))
	if err != nil {
		return 0, errors.New("readlink fail")
	}

	params := reg.FindStringSubmatch(link)
	if params == nil {
		return 0, errors.New("get ns fail")
	}

	ns, err := strconv.ParseUint(params[1], utils.DecimalBase, utils.Uint64Bits)
	if err != nil {
		return 0, err
	}

	return ns, nil
}

// Setns can the namespace of current process
func Setns(fd uintptr, flags uintptr) error {
	if _, _, err := syscall.RawSyscall(syscall.SYS_SETNS, fd, flags, 0); err != 0 {
		return errors.New("setns fail")
	}
	return nil
}

// NSEnter can enter the same net namespace of one specified process
func NSEnter(pid uint64) error {
	file, err := os.Open(fmt.Sprintf("%s/%d/ns/net", ProcDir, pid))
	if err != nil {
		log.Debugf("open pid %d ns file failed\n", pid)
		return err
	}
	defer file.Close()

	if err := Setns(file.Fd(), 0); err != nil {
		return err
	}
	return nil
}

// SetRootNs can return to the root namespace
func SetRootNs() error {
	return NSEnter(1)
}

// RemountSysfs can remount sysfs
func RemountSysfs() error {
	if err := syscall.Unmount("/sys", syscall.MNT_DETACH); err != nil {
		return err
	}
	if err := syscall.Mount("sysfs", "/sys", "sysfs", 0, ""); err != nil {
		return err
	}
	return nil
}

// MountSysfs can mount sysfs to specified path
func MountSysfs(path string) error {
	if err := syscall.Mount("sysfs", path, "sysfs", 0, ""); err != nil {
		return err
	}
	return nil
}

// UmountSysfs do the same as unmount
func UmountSysfs(path string) error {
	if err := syscall.Unmount(path, syscall.MNT_DETACH); err != nil {
		return err
	}
	return nil
}

// RemountNewSysfs remount sysfs to with specified path
func RemountNewSysfs(path string) error {
	if err := syscall.Unmount(path, syscall.MNT_DETACH); err != nil {
		return err
	}
	if err := syscall.Mount("sysfs", path, "sysfs", 0, ""); err != nil {
		return err
	}
	return nil
}
