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

// Package netaffinity provides functions for net affinity detection
package netaffinity

import (
	"bufio"
	"errors"
	"fmt"
	"gitee.com/wisdom-advisor/common/cpumask"
	"gitee.com/wisdom-advisor/common/utils"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const sysfsPerm = 0700
const (
	hexStrLenIPv4  = 8
	AssumedDevNum  = 10
	AssumedNodeNum = 4
	AssumedCPUNum  = 96
)

// ProcDir is the path of proc sysfs
var ProcDir = "/proc/"

// NetDevDir is the path of net device sysfs
var NetDevDir = "/sys/class/net/"

// NetInterface is to describe one net device
type NetInterface struct {
	IrqNode *[]int
	Name    string
	PCINode int
}

// NSIPDevCache is cache for IP inode pairs
type NSIPDevCache struct {
	Cache   *map[string]string
	NSName  string
	NSInode uint64
}

func getProcSocketInode(tid uint64) ([]uint64, error) {
	var inodes []uint64
	path := fmt.Sprintf("%s%d/fd/", ProcDir, tid)
	reg := regexp.MustCompile(`socket:\[(\d+)\]`)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return inodes, errors.New("read fd dir fail")
	}

	for _, file := range files {
		link, err := os.Readlink(path + file.Name())
		if err != nil {
			log.Debug("Socket Readlink fail")
			continue
		}
		params := reg.FindStringSubmatch(link)
		if params != nil {
			inode, err := strconv.ParseUint(params[1], utils.DecimalBase, utils.Uint64Bits)
			if err == nil {
				inodes = append(inodes, inode)
			}
		}
	}

	return inodes, nil
}

// CreateDevHashCache is to get cache that decribe net device existed
func CreateDevHashCache() (*map[string]string, error) {
	cache := make(map[string]string, AssumedDevNum)

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, dev := range interfaces {
		addrs, _ := dev.Addrs()
		if len(addrs) > 0 {
			ip, _, err := net.ParseCIDR(addrs[0].String())
			if err != nil {
				log.Error(err)
				continue
			}
			cache[ip.String()] = dev.Name
		}
	}
	return &cache, nil
}

func transNetOrderHexToIP(hexStr string) (net.IP, error) {
	var ret net.IP
	var ip [4]byte

	if len(hexStr) > hexStrLenIPv4 {
		reg := regexp.MustCompile("0000000000000000FFFF0000([A-Za-z0-9]+)")
		params := reg.FindStringSubmatch(string(hexStr))
		if params != nil {
			hexStr = params[1]
		} else {
			return nil, errors.New("not vaild IPv4 addr")
		}
	}
	for i := 0; i < 4; i++ {
		tmp := hexStr[i*2 : i*2+2]
		elem, err := strconv.ParseUint(tmp, 16, 8)
		if err != nil {
			return nil, err
		}
		ip[3-i] = byte(elem)
	}

	ret = net.IPv4(ip[0], ip[1], ip[2], ip[3])

	return ret, nil
}

// GetInodeIP is to get the related IP of one inode
func GetInodeIP(path string) (map[uint64]net.IP, error) {
	ret := make(map[uint64]net.IP, AssumedDevNum)
	reg := regexp.MustCompile(
		`\s+\S+\s+([A-Za-z0-9]+):\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+(\S+)`)

	file, err := os.Open(path)
	if err != nil {
		return ret, errors.New("open file fail")
	}
	defer file.Close()

	buf := bufio.NewReader(file)
	_, _, tag := buf.ReadLine()
	if tag == io.EOF {
		return ret, nil
	}

	for {
		line, _, tag := buf.ReadLine()
		if tag == io.EOF {
			break
		}
		params := reg.FindStringSubmatch(string(line))
		if params != nil {
			ip, err := transNetOrderHexToIP(params[1])
			if err != nil {
				log.Debug(err)
				continue
			}
			if ip.IsLoopback() {
				continue
			}
			inode, err := strconv.ParseUint(params[2], utils.DecimalBase, utils.Uint64Bits)
			if err != nil {
				log.Debug("Get inode error\n")
				continue
			}
			ret[inode] = ip
		}
	}
	return ret, nil
}

func getDevByIP(ip net.IP, cache map[string]string) string {
	if dev, ok := cache[ip.String()]; ok {
		return dev
	}
	return ""
}

func isPCIAddr(path string) bool {
	reg := regexp.MustCompile(
		`[a-zA-Z0-9]{4}:[a-zA-Z0-9]{2}:[a-zA-Z0-9]{2}.[a-zA-Z0-9]{1}`)
	params := reg.FindStringSubmatch(path)
	return params != nil
}

// GetNetDevPCIPath is to get the PCI path of one net device
func GetNetDevPCIPath(name string) (string, error) {
	ret := "/wisysfs/devices/"
	link, err := os.Readlink("/wisysfs/class/net/" + name)
	if err != nil {
		return "", errors.New("readlink fail")
	}
	reg := regexp.MustCompile(`.*\/devices\/(pci.*)`)

	params := reg.FindStringSubmatch(link)
	if params == nil {
		return "", errors.New("not a PCI device")
	}

	lines := strings.Split(params[1], "/")
	ret = ret + lines[0] + "/"
	lines = lines[1:]
	for _, line := range lines {
		if isPCIAddr(line) {
			ret = ret + line + "/"
		}
	}
	return ret, nil
}

// GetNetDevNUMANode is to get the preferred NUMA node of one net device
func GetNetDevNUMANode(name string) (int, error) {
	path, err := GetNetDevPCIPath(name)
	if err != nil {
		return -1, err
	}

	tmp, err := ioutil.ReadFile(path + "/numa_node")
	if err != nil {
		return -1, errors.New("read numa_node fail")
	}

	numa, err := strconv.ParseInt(strings.Replace(string(tmp), "\n", "", -1), utils.DecimalBase, utils.Uint64Bits)
	if err != nil {
		return -1, err
	}
	return int(numa), nil
}

func isIrqExisted(irq uint64) bool {
	path := fmt.Sprintf("/proc/irq/%d", irq)
	return utils.IsFileExisted(path)
}

func getNetDevIrq(dev string) ([]uint64, error) {
	var irqs []uint64

	path := NetDevDir + dev + "/device/msi_irqs/"
	if utils.IsFileExisted(path) {
		files, err := ioutil.ReadDir(path)
		if err != nil {
			return irqs, errors.New("read irq dir fail")
		}
		for _, file := range files {
			irq, err := strconv.ParseUint(file.Name(), utils.DecimalBase, utils.Uint64Bits)
			if err != nil {
				continue
			}
			if isIrqExisted(irq) {
				irqs = append(irqs, irq)
			}
		}
	}
	return irqs, nil
}

func getIrqCPUAffinity(irq uint64) ([]int, error) {
	var cpus []int
	var mask cpumask.Cpumask

	if !isIrqExisted(irq) {
		return cpus, errors.New("irq not exists")
	}
	path := fmt.Sprintf("%s/irq/%d/effective_affinity", ProcDir, irq)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return cpus, errors.New("read irq affinity fail")
	}
	affini := strings.Replace(string(data), ",", "", -1)
	affini = strings.Replace(affini, "\n", "", -1)

	mask.ParseString(affini)

	for cpu := mask.Foreach(-1); cpu != -1; cpu = mask.Foreach(cpu) {
		cpus = append(cpus, cpu)
	}
	return cpus, nil
}

func getDevIrqNode(dev string) (*[]int, error) {
	var nodes []int
	cpuMap := make(map[int]int, AssumedCPUNum)
	nodeMap := make(map[int]int, AssumedNodeNum)

	irqs, err := getNetDevIrq(dev)
	if err != nil {
		return nil, err
	}

	for _, irq := range irqs {
		cpus, err := getIrqCPUAffinity(irq)
		if err != nil {
			log.Info(err)
			continue
		}
		for _, cpu := range cpus {
			cpuMap[cpu] = cpu
		}
	}

	for key := range cpuMap {
		node := utils.GetCPUNumaID(key)
		nodeMap[node] = node
	}

	for key := range nodeMap {
		nodes = append(nodes, key)
	}
	return &nodes, nil
}

func getDevRelated(pid uint64, cache map[string]string) ([]string, error) {
	devMap := make(map[string]int, AssumedDevNum)
	var dev []string
	pathes := []string{"/net/tcp", "/net/tcp6", "/net/udp", "/net/udp6"}

	inodes, err := getProcSocketInode(pid)
	if err != nil {
		return dev, err
	}

	file, err := os.Open(fmt.Sprintf("%s/%d/ns/net", ProcDir, pid))
	if err != nil {
		return dev, errors.New("open ns file fail")
	}
	defer file.Close()

	for _, inode := range inodes {
		for _, path := range pathes {
			inodeMap, err := GetInodeIP(ProcDir + path)
			if err != nil {
				log.Info(err)
				continue
			}
			if ip, ok := inodeMap[inode]; ok {
				tmp := getDevByIP(ip, cache)
				if tmp != "" {
					devMap[tmp] = 1
					break
				}
			}
		}
	}

	for key := range devMap {
		dev = append(dev, key)
	}
	return dev, nil
}

func getDevCache(pid uint64) (*NSIPDevCache, error) {
	var cache NSIPDevCache
	var err error

	if cache.NSInode, err = GetPidNSInode(pid); err != nil {
		return nil, err
	}
	if cache.Cache, err = CreateDevHashCache(); err != nil {
		return nil, err
	}

	return &cache, nil
}

func getDevIndex(dev string) (int, error) {
	data, err := ioutil.ReadFile("/wisysfs/class/net/" + dev + "/ifindex")
	if err != nil {
		return -1, errors.New("open ifindex fail")
	}
	index, err := strconv.ParseInt(strings.Replace(string(data), "\n", "", -1),
		utils.DecimalBase, utils.Uint64Bits)
	if err != nil {
		return -1, err
	}
	return int(index), nil
}

func getDevLink(dev string) (int, error) {
	data, err := ioutil.ReadFile("/wisysfs/class/net/" + dev + "/iflink")
	if err != nil {
		return -1, errors.New("open iflink fail")
	}
	link, err := strconv.ParseInt(strings.Replace(string(data), "\n", "", -1),
		utils.DecimalBase, utils.Uint64Bits)
	if err != nil {
		return -1, err
	}
	return int(link), nil
}

func getIflink(dev string) int {
	index, err := getDevIndex(dev)
	if err != nil {
		log.Debug(err)
		return -1
	}
	link, err := getDevLink(dev)
	if err != nil {
		log.Debug(err)
		return -1
	}

	if link != index {
		return link
	}
	return -1
}

func getNetifInfo(name string) NetInterface {
	var dev NetInterface
	var err error
	dev.Name = name

	dev.IrqNode, err = getDevIrqNode(name)
	if err != nil {
		log.Info(err)
	}

	dev.PCINode, err = GetNetDevNUMANode(name)
	if err != nil {
		log.Info(err)
	}

	return dev
}

func getDevByIndex(index int) string {
	files, err := ioutil.ReadDir(NetDevDir)
	if err != nil {
		log.Debug("Read dir fail\n")
		return ""
	}
	for _, file := range files {
		data, err := ioutil.ReadFile(NetDevDir + file.Name() + "/ifindex")
		if err != nil {
			log.Debug("Get ifindex fail")
			continue
		}
		ifindex, err := strconv.ParseInt(strings.Replace(string(data), "\n", "", -1),
			utils.DecimalBase, utils.Uint64Bits)
		if err != nil {
			log.Debug("Get ifindex fail")
			continue
		}
		if int(ifindex) == index {
			return file.Name()
		}
	}
	return ""
}

func isBondingDev(dev string) bool {
	return utils.IsFileExisted(NetDevDir + dev + "/bonding")
}

func getBondSlaves(dev string) []string {
	var devs []string
	tmp, err := ioutil.ReadFile(NetDevDir + dev + "/bonding/slaves")
	if err != nil {
		log.Debug("Get Slaves fail")
		return devs
	}
	return strings.Split(strings.Replace(string(tmp), "\n", "", -1), " ")

}

func preDetect() error {
	if err := os.Mkdir("/wisysfs", sysfsPerm); err != nil {
		return errors.New("make dir fail")
	}
	if err := MountSysfs("/wisysfs"); err != nil {
		return errors.New("mount dir fail")
	}
	return nil
}

func postDetect() error {
	if err := UmountSysfs("/wisysfs"); err != nil {
		return errors.New("umount sysfs fail")
	}
	if err := os.RemoveAll("/wisysfs"); err != nil {
		return errors.New("clean sysfs fail")
	}
	return nil
}

func detectDevRelated(pid uint64) ([]NetInterface, error) {
	if err := preDetect(); err != nil {
		return nil, err
	}

	ret, err := GetRealDevRelated(pid)
	if err != nil {
		return nil, err
	}

	if err = postDetect(); err != nil {
		log.Error(err)
	}
	return ret, err
}

// GetRealDevRelated  is to get the related net device of one process
func GetRealDevRelated(pid uint64) ([]NetInterface, error) {
	var devInfo []NetInterface
	var linkArr []int
	var hostDevs Queue

	NSEnter(pid)
	if err := RemountNewSysfs("/wisysfs"); err != nil {
		SetRootNs()
		return devInfo, err
	}

	cache, err := getDevCache(pid)
	if err != nil {
		SetRootNs()
		return devInfo, err
	}

	devs, err := getDevRelated(pid, *(cache.Cache))
	if err != nil {
		SetRootNs()
		return devInfo, err
	}

	for _, dev := range devs {
		if iflink := getIflink(dev); iflink != -1 {
			linkArr = append(linkArr, iflink)
		} else {
			devInfo = append(devInfo, getNetifInfo(dev))
		}
	}
	SetRootNs()
	if err := RemountNewSysfs("/wisysfs"); err != nil {
		return devInfo, err
	}

	for _, link := range linkArr {
		dev := getDevByIndex(link)
		if dev == "" {
			log.Debug("Fail to get link dev\n")
		} else {
			hostDevs.PushBack(dev)
		}
	}

	for {
		if hostDevs.IsEmpty() {
			break
		}
		dev, _ := hostDevs.PopFront()
		if iflink := getIflink(dev.(string)); iflink != -1 {
			hostDevs.PushBack(getDevByIndex(iflink))
		} else if isBondingDev(dev.(string)) {
			slaves := getBondSlaves(dev.(string))
			for _, slave := range slaves {
				hostDevs.PushBack(slave)
			}
		} else {
			devInfo = append(devInfo, getNetifInfo(dev.(string)))
		}
	}
	return devInfo, nil
}

// GetProcessNetNuma is to get the preferred NUMA node of one process according to net access
func GetProcessNetNuma(pid uint64) (int, error) {
	var numaNode int
	var SecNode int
	nodes := make(map[int]int, AssumedNodeNum)
	nodesBak := make(map[int]int, AssumedNodeNum)

	devs, err := detectDevRelated(pid)
	if err != nil {
		return -1, err
	}

	log.Debugf("%d related numa\n", pid)
	for _, dev := range devs {
		log.Debugf("    %s\n", dev.Name)
		for _, irqNode := range *(dev.IrqNode) {
			nodes[irqNode] = 1
			log.Debugf("    irqnode: %d\n", irqNode)
			numaNode = irqNode
		}
		if dev.PCINode > 0 {
			nodesBak[dev.PCINode] = dev.PCINode
			SecNode = dev.PCINode
		}
	}
	if len(nodes) != 1 {
		log.Info("Net dev IRQs on multi numa nodes or unable to handle irq info\n")
		if len(nodesBak) == 1 {
			return SecNode, nil
		}
		return -1, nil
	}
	return numaNode, nil
}
