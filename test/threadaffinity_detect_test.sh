#!/bin/sh
# Copyright (c) 2020 Huawei Technologies Co., Ltd.
#
# wisdom-advisor is licensed under the Mulan PSL v2.
# You can use this software according to the terms and conditions of the Mulan PSL v2.
# You may obtain a copy of Mulan PSL v2 at:
#     http://license.coscl.org.cn/MulanPSL2
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v2 for more details.
# Create: 2020-6-9

function do_test() {
	local pid
	local group

	export GOPATH=`cd ../../../../;pwd`
	
	"$GOPATH"/pkg/wisdomd --printlog --loglevel debug &
	pid=`echo $!`
	sleep 5
	"$GOPATH"/pkg/wisdom --printlog --loglevel debug threadsaffinity --percore --task sem  --tracetime=5 --period=10
	if [  "$pid"x == ""x ]; then
		echo "start wisdomd fail"
		rm tmp.log
		return 1
	fi
	echo "wisdomd: $pid"
	./common/sem
	kill -2 "$pid"
	
	wait
	group=`grep "Bind group" tmp.log`
	if [ "$group" == "" ]; then
		echo "Can't get bind group"
		return 1
	fi

	rm tmp.log
	return 0
}

do_test
