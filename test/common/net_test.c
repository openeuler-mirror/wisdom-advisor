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
#include <unistd.h>
#include <memory.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <sys/prctl.h>
#include <ifaddrs.h>

#define PORT 9999
#define MAXSIZE 100

int server(struct sockaddr_in *saddr)
{
	char buffer[MAXSIZE] = "nothing";
	int sk, i, conn, res;
	struct sockaddr_in sin;
	struct sockaddr_in cli;
	socklen_t len = sizeof(struct sockaddr);

	sk = socket(AF_INET, SOCK_STREAM, 0);
	if (sk == -1) {
		printf("Create socket error.\n");
		return -1;
	}
	
	memset(&sin, 0, sizeof(sin));
	sin.sin_family = AF_INET;
	sin.sin_port = htons(PORT);
	sin.sin_addr = saddr->sin_addr;
	
	res = bind(sk, (struct sockaddr *)&sin, sizeof(struct sockaddr));
	if (res == -1) {
		printf("Bind socket fail.\n");
		close(sk);
		return -1;
	}

	res = listen(sk, 1);
	if (res == -1) {
		printf("Listen fail.\n");
		close(sk);
		return -1;
	}
	conn = accept(sk, (struct sockaddr *)&cli, &len);
	if (conn == -1) {
		printf("Accept fail\n");
		close(sk);
		return -1;
	}

	for (i = 0; i < 30; i++) {
		res = write(conn, buffer, strlen(buffer));
		if (res == -1) {
			break;
		}
		sleep(1);
	}
	close(conn);
	close(sk);
	return 0;
}

int client(struct sockaddr_in *saddr)
{
	char buffer[MAXSIZE] = {0};
	int sk, i, ret;
	struct sockaddr_in sin;

	sk = socket(AF_INET, SOCK_STREAM, 0);
	if (sk == -1) {
		printf("Create socket error.\n");
		return -1;
	}
	
	memset(&sin, 0, sizeof(sin));
	sin.sin_family = AF_INET;
	sin.sin_addr = saddr->sin_addr;
	sin.sin_port = htons(PORT);
	ret = connect(sk, (struct sockaddr *)&sin, sizeof(sin));
	if (ret == -1) {
		printf("Connect fail\n");
		close(sk);
		return -1;
	}

	for (i = 0; i < 30; i++) {
		ret = read(sk, buffer, MAXSIZE);
		if (ret == -1) {
			close(sk);
			return 0;
		}
	}
	close(sk);
	return 0;
}

int main()
{
	pid_t pid;
	int ret;

	struct sockaddr_in *saddr = NULL;
	struct ifaddrs *ifa, *ifList;

	if (getifaddrs(&ifList) < 0) {
		return -1;
	}

	for (ifa = ifList; ifa != NULL; ifa = ifa->ifa_next) {
		if(ifa->ifa_addr->sa_family == AF_INET && 0 != strcmp(ifa->ifa_name,"lo")) {
			saddr = (struct sockaddr_in *)ifa->ifa_addr;
			printf("%s\n", inet_ntoa(saddr->sin_addr));
			break;
		}
	}
	if (saddr == NULL) {
		printf("No valid netif");
		return -1;
	}

	pid = fork();
	if (pid == 0) {
		sleep(2);
		ret = client(saddr);
		if (ret != 0) {
			printf("Start client fail");
			return -1;
		}
	} else {
		ret = server(saddr);
		if (ret != 0) {
			printf("Start server fail");
			return -1;
		}
	}
	return 0;
}
