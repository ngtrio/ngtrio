---
title: Cap'n Proto
date: 2021-11-14 16:52:38
tags:
- RPC
---
从Kitex的Rodemap中看到这一条：

![image-20211115000303622](image-20211115000303622.png)

### 简介

[Cap'n Proto ](https://capnproto.org/index.html)是Protocol Buffer 2 的主要作者 Kenton Varda 经过多年的实践经验和听取用户使用建议后所设计出来的数据交换格式(interchange format) 和 RPC 系统。

### 特性

- **增量读：** Cap’n Proto 的 message 不用等全部接收完成了才开始处理。因为消息中的 inner 对象是被安排在 outer 对象的后面的，而不像其他大多协议一样是嵌套关系。
- **随机访问：**能够只读一条消息中的一个 field，而不用解析整个消息。
- **mmap：**支持mmp。像 protobuf 就不支持，读很小一部分信息也会加载所有数据到用户态。（其实也没啥可比性，这个特性主要还是因为没有encoding/decoding过程）
- **高效的进程间通信：** 同机器上的多个进程可以通过共享内存来分享Cap'n Message。没有必要将数据在user/kernel之间来回拷贝。
- **内存集中分配：**  Cap’n Proto 的对象通常会集中分配，有点类似池化，达到可复用的效果，缓存友好。
- **生成的代码量小：** Protobuf 会为每种消息类型生成解析和编码的代码,代码量巨大. Cap’n Proto 生成的代码少至少一个数量级。
- **Time-traveling RPC：**下面介绍。

### RPC Protocol

[Time travel](https://capnproto.org/rpc.html) 



​	

