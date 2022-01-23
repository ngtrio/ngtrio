---
title: OpenWRT跑Go定时任务
date: 2022-01-23 10:48:19
tags: 
- OpenWRT 
- Go
---
## 背景
因为前段时间为了做一个全局的🪜，于是把家里的路由器刷了一个OpenWRT，尽管路由器性能比较拉胯，但是架起了🪜后，还是有一定的闪存空间以及内存空间，最近想把这点空间也给利用上，~~路由器24h工作的电费得找补回来。~~    
想到之前学习Go的时候写了个简单的TG机器人，刚好最近想跑几个定时任务，于是决定把这个机器人搭建到路由器上。

## 交叉编译
因为路由器的CPU架构不一样，我自己本机编译的二进制自然是无法在路由器上跑起来的。所以需要进行交叉编译，这里看到我的路由器CPU架构是`mips`  
```
root@OpenWrt:~# uname -m
mips
```
[Go原生就支持交叉编译`mips`指令集的二进制](https://go.dev/doc/install/source#environment)，所以这一步就很简单了。   
```sh
GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build
```
## 时区问题
晚上定时任务写好，编译好，在路由器上跑起来后就睡觉去了，但是第二天早上九点多起床发现我的tg bot并没有收到cron为`0 0 8 * * ?`的定时任务消息。  
首先ssh到路由器，可以确定我的tgbot的进程并没有挂掉，查看日志也没有任何错误信息，但是发现日志的时间都是`UTC`时间，这就意味着定时任务其实还没到触发的时间。   
但是记得之前是在OpenWRT后台是设置过`UTC+8`时区的，并且在路由器的shell执行`date -R`也显示的是`UTC+8`   
```
root@OpenWrt:/etc# date -R
Sun, 23 Jan 2022 10:56:06 +0800
```   
而我也清楚记得Go的time.Format()默认就是采用的系统时区，这里Go没有读取到就有点奇怪了。   
于是了解了下细节：   
1. OpenWRT在哪里设置了时区   
    * /etc/TZ文件
2. Go从哪里获取系统时区
    * 读取TZ环境变量
    * 读取/etc/localtime文件
    * 本地时区读取失败，使用 UTC 时间   

问题就很显然了。   
解决：   
```sh
opkg update
opkg install zoneinfo-asia
ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
```
## 参考资料
[Writing and Compiling Program For OpenWrt](https://stackoverflow.com/a/60161561/11571735)   
[Go environment](https://go.dev/doc/install/source#environment)   
[深入理解GO时间处理(time.Time)](https://www.imhanjm.com/2017/10/29/%E6%B7%B1%E5%85%A5%E7%90%86%E8%A7%A3golang%E6%97%B6%E9%97%B4%E5%A4%84%E7%90%86(time.time)/)
