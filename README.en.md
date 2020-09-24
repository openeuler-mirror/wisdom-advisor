## Introduction
Wisdom-advisor is a tunning framework aimming at improving the performance of applications using scheduling or
other methods.
Three policy is supported now in wisdom-advisor:
1. Thread affinity specified by users: parse __SCHED_GROUP__ to get thread affinity.
2. Thread affinity detection: trace syscall futex to get thread affinity.
3. Thread grouping: trace net and IO syscall, partition threads by user define.

There are several functions optinal that assist scheduling decision like NUMA affinity detection which can
reduce access cross-NUMA memory, net affinity detection which can detect net accessing processes
and get the perferred NUMA node according to the net device they use and more.

Wisdom-advisor now support linux on x86 and arm64 architectrue.
Wisdom-advisor should run with root privileges.
## Build
Please note that go environment is needed and one accessible goproxy server is necessary for Go Modules is used here to manage vendoring packages.

To set available proxy, please refer to [Go Module Proxy](https://proxy.golang.org)
```
mkdir -p $GOPATH/src/gitee.com
cd $GOPATH/src/gitee.com
git clone <wisdom-advisor project>
cd wisdom-advisor
export GO111MODULE=on
go mod vendor
make
```
wisdomd binary file is in $GOPATH/pkg/

Run testcases
```
make check
```
## Install
In wisdom-advisor project directory.
```
make install
```
## How to use
wisdomd is the daemon and wisdom is the client.
Get help infomation
```
wisdomd -h
wisdom -h
```
When using thread affinity policy without automatic detection. Wisdomd will get group information from /proc/pid/envrion
and auto set affinity for threads in group. Group environment variable format is as below:
\_\_SCHED\_GROUP\_\<group\_name\>=thread\_name1,thread\_name2...
```
wisdom usersetaffinity 
```
Or we can use automatic detection.
```
wisdom threadsaffinity --task sem 
```
When using thread grouping, IO cpu list and net cpu list should be provided.
```
wisdom threadsgrouping --task test --IO 1-2,5,6 --net 3-4
```
Wisdomd will do some scanning when using threadsaffinity policy with automatic detection and threadsgrouping policy and
this scanning opertation can be shutdown or restart.
```
wisdom scan stop
```
Other options can be found in help information.

## Licensing
Wisdom is licensed under the Mulan PSL v2.
