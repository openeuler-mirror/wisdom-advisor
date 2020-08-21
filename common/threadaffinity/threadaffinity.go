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

// Package threadaffinity provides functions to detect threads affinity
package threadaffinity

import (
	"fmt"
	"gitee.com/wisdom-advisor/common/ptrace"
	"gitee.com/wisdom-advisor/common/utils"
	log "github.com/sirupsen/logrus"
	"sort"
)

type groupInfo struct {
	member []uint64
}

// TidsGroup describe tids which should be scheduled together
type TidsGroup struct {
	Tids      []uint64
	size      int
	GroupName string
}

var resultTidsSlice []TidsGroup
var resultGroupNum int
var resultTidsGroupAll []uint64

var resultOldTidsSlice []TidsGroup
var resultOldGroupNum int
var resutlOldTidsGroupAll []uint64

var resultListMap map[int]*groupInfo // the group info
var resultTidMap = make(map[uint64]map[uint64]int)
var resultTidArray map[int]uint64 // map tid and union set index
var currentPid uint64

func backupTids(backupEle *TidsGroup, ele TidsGroup) {
	(*backupEle).size = ele.size
	(*backupEle).GroupName = ele.GroupName
	copy((*backupEle).Tids, ele.Tids)
}

func backupTidsGroup() {
	resultOldGroupNum = resultGroupNum
	resultOldTidsSlice = make([]TidsGroup, 0)
	for i := 0; i < resultOldGroupNum; i++ {
		var tids TidsGroup
		backupTids(&tids, resultTidsSlice[i])
		resultOldTidsSlice = append(resultOldTidsSlice, tids)
	}
	resutlOldTidsGroupAll = make([]uint64, len(resultTidsGroupAll))
	copy(resutlOldTidsGroupAll, resultTidsGroupAll)
}

// PidChanged indicate the change of target process
func PidChanged(newPid uint64) (uint64, bool) {
	var changed bool
	var oldPid uint64
	if currentPid == newPid {
		changed = false
	} else {
		oldPid = currentPid
		currentPid = newPid
		changed = true
		createTidMap()

	}
	return oldPid, changed
}

func createGroupList() {
	resultListMap = make(map[int]*groupInfo)
	resultTidsSlice = make([]TidsGroup, 0)
}

func createTidMap() {
	resultTidMap = make(map[uint64]map[uint64]int)
}

func createTidArrayMap() {
	resultTidArray = make(map[int]uint64)
}

func initVariable() {
	createGroupList()
	createTidArrayMap()
}

func mapConnect(mapA map[uint64]int, mapB map[uint64]int) bool {
	for key := range mapA {
		if mapB[key] != 0 {
			return true
		}
	}
	return false
}

func initUnion(arrayLen int, array []int) {
	for i := 0; i < arrayLen; i++ {
		array[i] = i
	}
}

func findRoot(son int, array []int) int {
	temp := son
	for {
		if array[temp] == temp {
			return temp
		}
		temp = array[temp]
	}
}

func union(a int, b int, array []int, count *int) {
	rootA := findRoot(a, array)
	rootB := findRoot(b, array)
	if rootA == rootB {
		return
	}
	array[rootA] = rootB
	*count--
}

func getGroups(tidArray map[int]uint64, array []int, arrayLen int) {
	var ok bool
	for i := 0; i < arrayLen; i++ {
		root := findRoot(i, array)
		if _, ok = resultListMap[root]; !ok {
			var ele groupInfo
			resultListMap[root] = &ele
		}
		resultListMap[root].member = append(resultListMap[root].member, tidArray[i])
	}
}

func groupTids(pidMap map[uint64]map[uint64]int, tidArray map[int]uint64) {
	mapLen := len(tidArray)
	array := make([]int, mapLen, mapLen)
	count := mapLen
	initUnion(mapLen, array)
	for i := 0; i < mapLen; i++ {
		tempMapA := pidMap[tidArray[i]]
		for j := i + 1; j < mapLen; j++ {
			tempMapB := pidMap[tidArray[j]]
			if mapConnect(tempMapA, tempMapB) {
				union(i, j, array, &count)
			}
		}
	}
	getGroups(tidArray, array, mapLen)
}

func getGroupName(tidGroup TidsGroup, name string) string {
	var suffix string
	for tid := range tidGroup.Tids {
		suffix = fmt.Sprintf("%s_%d", suffix, tid)
	}
	groupName := fmt.Sprintf("%s%s", name, suffix)
	return groupName
}

func mapToList(listMap map[int]*groupInfo, tidsSlice *[]TidsGroup, name string) int {
	var lenList int
	for _, val := range listMap {
		ele := TidsGroup{val.member, len(val.member), ""}
		*tidsSlice = append(*tidsSlice, ele)
	}
	sort.Slice(*tidsSlice, func(i, j int) bool {
		return (*tidsSlice)[i].size > (*tidsSlice)[j].size // 降序
	})
	for _, m := range *tidsSlice {
		if m.size > 1 {
			m.GroupName = getGroupName(m, name)
			lenList++
		} else {
			return lenList
		}
	}
	return lenList
}

func collectAllGroupTids(tidsSlice []TidsGroup, num int, tidsGroupAll *[]uint64) {
	*tidsGroupAll = make([]uint64, 0)
	for i := 0; i < num; i++ {
		*tidsGroupAll = append(*tidsGroupAll, tidsSlice[i].Tids...)
	}
}

// StartGroups is to start detect threads affinity
func StartGroups(name string, tracetime int) {
	var index = 0

	initVariable()

	pid, err := utils.GetPid(name)
	if err != nil {
		log.Info("Get pid fail\n")
		return
	}

	infos, err := ptrace.DoCollect(pid, tracetime, ptrace.FutextDetect)
	if err != nil {
		log.Error(err)
		return
	}

	for _, info := range infos {
		resultTidMap[info.Pid] = info.SysCount.FutexMap
		resultTidArray[index] = info.Pid
		index++
	}

	groupTids(resultTidMap, resultTidArray)
	resultGroupNum = mapToList(resultListMap, &resultTidsSlice, name)
	collectAllGroupTids(resultTidsSlice, resultGroupNum, &resultTidsGroupAll)
}

// GetTidSlice is to get the threads group that has been recognized
func GetTidSlice() ([]TidsGroup, int) {
	return resultTidsSlice, resultGroupNum
}

// GetOldTidSlice is to get the previous threads group
func GetOldTidSlice() ([]TidsGroup, int) {
	return resultOldTidsSlice, resultOldGroupNum
}

func equalSliceUint64(a []uint64, b []uint64) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func tidsChanged() bool {
	sort.Slice(resultTidsGroupAll, func(i, j int) bool { return resultTidsGroupAll[i] < resultTidsGroupAll[j] })
	sort.Slice(resutlOldTidsGroupAll, func(i, j int) bool { return resutlOldTidsGroupAll[i] < resutlOldTidsGroupAll[j] })
	return !equalSliceUint64(resutlOldTidsGroupAll, resultTidsGroupAll)
}

// GroupChanged indicate whether the group recognized has changed
func GroupChanged() bool {
	if resultGroupNum != resultOldGroupNum {
		log.Info("group_num changed\n")
		backupTidsGroup()
		return true
	}
	if tidsChanged() {
		log.Info("tids changed\n")
		backupTidsGroup()
		return true
	}
	return false
}
