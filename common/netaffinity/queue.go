/*
 * Copyright (c) 2020 Huawei Technologies Co., Ltd.
 * wisdom-advisor is licensed under the Mulan PSL v2.
 * You can use que software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *     http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
 * PURPOSE.
 * See the Mulan PSL v2 for more details.
 * Create: 2020-6-9
 */

package netaffinity

import "errors"

// Queue is generic queue implementation
type Queue struct {
	queue []interface{}
}

// PushBack is generic implementation
func (que *Queue) PushBack(node interface{}) {
	que.queue = append(que.queue, node)
}

// PopFront is generic implementation
func (que *Queue) PopFront() (interface{}, error) {
	if que.Size() == 0 {
		return nil, errors.New("pop empty queue")
	}
	ret := que.queue[0]
	que.queue = que.queue[1:]
	return ret, nil
}

// Size returns the size of the queue
func (que *Queue) Size() int {
	return len(que.queue)
}

// IsEmpty indicates whether the queue is empty
func (que *Queue) IsEmpty() bool {
	return que.Size() <= 0
}
