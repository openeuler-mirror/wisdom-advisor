## Introduction

Wisdom-advisor is a tuning framework that uses scheduling or other methods to improve the performance of applications.  

Wisdom-advisor supports the three policies:  

1. Thread affinity specified by users: parses __SCHED_GROUP__ to get thread affinity.  
2. Thread affinity detection: traces syscall futex to get thread affinity.  
3. Thread grouping: detects and binds threads to network and I/O CPUs based on users' settings.  

There are several tuning policies available. For example, NUMA affinity detection can reduce access cross-NUMA memory. Another example is network affinity detection, which can detect network access processes and obtain the preferred NUMA node according to the used network devices.  

Wisdom-advisor now supports Linux on x86 and ARM64.  

Wisdom-advisor requires the root privileges.  

## Build

Please note that the Go environment is needed and one accessible Goproxy server is necessary for using Go Modules to manage vendoring packages.  

To set an available proxy, please refer to [Go Module Proxy](https://proxy.golang.org).  

```
mkdir -p $GOPATH/src/gitee.com
cd $GOPATH/src/gitee.com
git clone <wisdom-advisor project>
cd wisdom-advisor
export GO111MODULE=on
go mod vendor
make
```

Wisdomd binary file are saved in the **$GOPATH/pkg/** directory.  

Run test cases:  

```
make check
```

## Install

In the Wisdom-advisor project directory,  

```
make install
```

## How to use

Wisdomd is the daemon and Wisdom is the client.  
Get help information:  

```
wisdomd -h
wisdom -h
```

When using a thread affinity policy without automatic detection, Wisdomd gets group information from **/proc/pid/envrion**
and automatically sets affinity for threads in the group. Group environment variables are in the following format:  
\_\_SCHED\_GROUP\_\<group\_name\>=thread\_name1,thread\_name2...  

```
wisdom usersetaffinity 
```

Alternatively, we can use automatic detection:  

```
wisdom threadsaffinity --task sem 
```

When using thread grouping, the I/O CPU list and network CPU list should be provided.  

```
wisdom threadsgrouping --task test --IO 1-2,5,6 --net 3-4
```

Wisdomd will execute a scan when using a thread affinity policy with automatic detection or a thread grouping policy. This scan operation can be stopped or restarted.  

```
wisdom scan stop
```

Other options can be found in help information.  

## Licensing

Wisdom is licensed under Mulan PSL v2.