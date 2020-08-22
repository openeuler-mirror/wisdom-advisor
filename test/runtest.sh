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

array=(	"threadaffinity_detect_test.sh" 
	"threadgrouping_test.sh")
RET=0
for file in ${array[@]}
do
	sh -x $file
	if [ $? != 0 ]; then
		echo "case $file fail"
		RET=-1
		break
	fi
done

if [ $RET == -1 ]; then 
	echo "FAIL"
else
	echo "PASS"
fi
