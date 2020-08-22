## Introduction
Wisdom-advisor is a tunning framework aimming at improving the performance of applications using scheduling or
other methods.
Two policy is supported now in wisdom-advisor:
1. Thread affinity: schedule threads according to their affinity(the affinity can be specified by users or automatic detection).
2. Thread grouping: scheduling threads according to what they are doing.

There are several functions optinal that assist scheduling decision like NUMA affinity detection which can
reduce access cross-NUMA memory, net affinity detection which can detect net accessing processes
and get the perferred NUMA node according to the net device they use and more.

Wisdom-advisor now support arm64 architectrue, support for x86 is on the way.
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
Get help infomation
```
wisdomd -h
```
When using thread affinity policy without automatic detection. Wisdomd will get group information from /proc/pid/envrion
and auto set affinity for threads in group. Group environment variable format is as below:
\_\_SCHED\_GROUP\_\<group\_name\>=thread\_name1,thread\_name2...
```
wisdomd --policy threadsaffinity 
```
Or we can use automatic detection.
```
wisdomd --policy threadsaffinity --affinityAware
```
When using thread grouping, CPU partition description json script should be provided.
```
wisdomd --policy threadsgrouping --json XXX.json
```
Wisdomd will do some scanning when using threadsaffinity policy with automatic detection and threadsgrouping policy and
this scanning opertation can be shutdown or restart.
```
wisdom --scan start
wisdom --scan stop
```
Other options can be found in help information.

Note: 
For security consideration, the json script that describe CPU partition should be set with appropriate umask.
Normal users should not have the wirte or access permissions.
When not necessary, scan should be stop.
## Licensing
Wisdom is licensed under the Mulan PSL v2.
