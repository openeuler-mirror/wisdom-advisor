# wisdom智能感知调度

## 背景

服务器多数支持多路CPU，CPU tope支撑NUMA架构，对于CPU来说，跨P访问内存时延是最大的，跨die访问次之，传统Linux内核的调度算法默认认为所有节点是一样的，负载均衡和调度策略都是基于此来做的，这对arm64来说不太实用，从前期的实践来说，通过人工绑核手段可以显著的提升性能，但是这种方式缺乏灵活性。

方案实现思路

由于arm64 上性能的瓶颈主要是跨P访问内存带来的开销，因此解决性能问题的主要从以下出发点来考虑

限制进程的跨P迁移，确保进程在一个节点内部，并且从效率上来讲，尽量不要做迁移; 

减少进程的切换频率，降低切换开销; 

操作系统考虑负载和平衡的时候，更多的是从动态的角度来考虑，也不考虑业务特点，实际上在大多数业务场景，cpu的负载和业务模型都是固定的，并不需要频繁的迁移和做均衡，只要CPU没有成为瓶颈，业务也没有必要的去做负载的平衡和均衡。

本文优先考虑基于一种应用来达到优化的目的，来实现cpu资源管理策略机制，避免用户修改代码或者通过手动绑核的方式去优化的目的，为适配更多应用做准备。


## 功能简介

wisdom是一个智能调整框架，旨在使用调度或其他方法来提高应用程序的性能。wisdom现在支持两种策略：

1. 线程亲和性调度：根据线程的亲和性来绑核（亲和性可以由用户指定或自动检测）。
2. 线程分组调度：根据线程功能分组绑核。

有多种可选优化策略，例如NUMA亲和性检测可以减少跨NUMA内存的访问，网络亲和性检测可以检测网络访问进程并根据其使用的网络设备获取首选的NUMA节点，等等。

wisdom现在支持arm64，即将支持x86。wisdom必须以root特权运行。

## 调度调整策略

### 线程亲和性调度

#### 用户指定

在进程环境变量中配置\_\_SCHED_GROUP\_\_，Wisdomd将从/ proc / pid / envrion获取组信息，例如"\_\_SCHED_GROUP\_\_<group_name>=thread_name1,t",wisdom会根据\_\_SCHED_GROUP\_\_的配置来进行绑核

```shell
wisdomd --policy threadsaffinity 
```

#### 自动检测

wisdom会通过ptrace检测futex锁的关系，来推测哪些线程具有亲和性，将这些线程绑定在同一NUMA

```shell
wisdomd --policy threadsaffinity --affinityAware
```

#### NUMA亲和性检测

通过启动项task_faulting=1开启页面访问统计，wisdom会通过访问/proc/$tid/task/$tid/task_fault_siblings得到页面访问统计信息，然后将线程绑定到与其相关内存最多的numa

```shell
wisdomd --policy threadsaffinity --autonuma
```

#### CCL粒度绑定

```shell
wisdomd --policy threadsaffinity --cclaware
```

#### 非单核绑定（粗粒度绑定）

```shell
wisdomd --policy threadsaffinity --coarsegrain
```

### 线程分组调度

wisdom通过ptrace检测线程属于网络线程还是磁盘IO线程，根据用户指定的json文件，绑定指定的核

使用方法：

```shell
wisdomd --policy threadsgrouping --json XXX.json
```

json模板

```json
The script is like:
{
        "io": [
                "0-4",
                "93-96"
        ],
        "net": [
                "48-92",
                "5-47"
        ]
}
```

