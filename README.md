## 功能简介

wisdom是一个智能调整框架，旨在使用调度或其他方法来提高应用程序的性能。wisdom现在支持三种策略：

1. 用户指定的线程亲和性调度：解析\_\_SCHED_GROUP\_\_以获取线程亲和性。
2. 线程亲和性检测：跟踪syscall futex以获取线程亲和性。
3. 线程分组：按用户定义探测并绑定线程到net和IO CPU。

有多种可选优化策略，例如NUMA亲和性检测可以减少跨NUMA内存的访问，网络亲和性检测可以检测网络访问进程并根据其使用的网络设备获取首选的NUMA节点，等等。

wisdom现在支持linux下arm64和x86两种架构。

## 编译

```
mkdir -p $GOPATH/src/gitee.com
cd $GOPATH/src/gitee.com
git clone <wisdom-advisor project>
cd wisdom-advisor
export GO111MODULE=on
go mod vendor
make
```
编译出的二进制执行文件路径 $GOPATH/pkg/

运行测试用例
```
make check
```
## 安装
```
make install
```
## 如何使用
wisdomd是守护进程，wisdom是客户端。
获取帮助信息
```
wisdomd -h
wisdom -h
```
在进程环境变量中配置\_\_SCHED_GROUP\_\_，Wisdomd将从/ proc / pid / envrion获取组信息，例如"\_\_SCHED_GROUP\_\_<group_name>=thread_name1,t",wisdom会根据\_\_SCHED_GROUP\_\_的配置来进行绑核

```
wisdom usersetaffinity 
```
wisdom会通过ptrace检测futex锁的关系，来推测哪些线程具有亲和性，将这些线程绑定在同一NUMA

```shell
wisdom threadsaffinity --task sem 
```
使用线程分组时，应提供IO cpu列表和网络cpu列表
```
wisdom threadsgrouping --task test --IO 1-2,5,6 --net 3-4
```
Wisdomd在使用带有自动检测和线程分组策略时将执行一些扫描，此扫描操作可以关闭或重新启动。
```
wisdom scan stop
```
其他选项可以在帮助信息中找到。

## 许可证
Wisdom 许可证是根据木兰PSL v2授权的。