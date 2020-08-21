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

#include <stdio.h>
#include <pthread.h>
#include <semaphore.h>
#include <unistd.h>

typedef struct {
        sem_t semlock;
        int val;
} SemVal;

static void *SemAccessThread(void *arg)
{
	int i;
        SemVal *tempSemVal = (SemVal *)arg;
        sem_t * temp = &(tempSemVal->semlock);
	for (i = 0; i < 30; i++) {
                sem_wait(temp);
		sleep(1);
                sem_post(temp);
		sleep(1);
        }
	return NULL;
}

int main()
{
	SemVal sem[2];
        pthread_t thread[8];
        int threadResult;
	int i;

        sem_init(&(sem[0].semlock), 0, 1);
        sem_init(&(sem[1].semlock), 0, 1);

	for (i = 0; i < 8; i++) {
		threadResult = pthread_create(&thread[i], NULL, SemAccessThread, (void*)(&sem[i%2]));
		if (threadResult != 0) {
			printf("Create threads fail\n");
			return threadResult;
		}
	}
	
	sleep(30);

        return 0;
}
