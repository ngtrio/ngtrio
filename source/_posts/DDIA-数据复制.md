---
title: 读DDIA-数据复制
date: 2021-10-02 11:29:06
tags:
- 分布式
---
## 数据复制

分布式系统中， 通过数据复制，我们希望达到下列目的：

- 在全球各地数据中心间进行数据复制，使得数据在地理上更加接近用户，降低用户请求延迟
- 提高可用性，当一个节点出现故障，我们有经过数据复制得到的副本继续工作
- 横向扩展，用多个数据一致的节点提供同一个服务，提高吞吐量

目前业界有下面几种数据复制方案：

1. 主从复制
2. 多主节点复制
3. 无主节点复制

### 主从复制

原理简述：

1. 指定一个节点为主节点，其他都为从节点，客户端写数据库请求全部路由到主节点，由主节点首先将数据写入本地
2. 主节点数据本地写入完成后，将数据更改日志或者更改流发送给所有的从节点，从节点将数据写入本地，同时严格保持与主节点相同的写入数据
3. 客户端读数据请求可以路由到全部节点

**注意：从节点只接受读数据请求，写请求由主节点负责**

![](ddia-1.png)

图1. 基于领导者(主-从)的复制

**同步复制 or 异步复制**

![](ddia-2.png)

图2. Follower 1同步复制节点，Follower 2异步复制节点

- 同步复制
    
    用户的一次数据写请求在所有同步复制的从节点被写入后才会得到响应。
    
    - 优点：始终保证主从节点中的数据一致性
    - 缺点：只要有任何同步复制节点性能降低甚至故障，用户请求响应时间将大幅增加
- 异步复制
    
    用户的一次数据写请求在主节点写入后就会得到响应，主从复制将在后续异步进行
    
    - 优点：即使从节点出现数据复制滞后，主节点依旧能够响应写请求，吞吐量得到保证
    - 缺点：万一主节点主线故障下线，数据可能还没来得及复制完毕，导致新上线的主节点（从节点继承）数据丢失
- 半同步
    
    在业界实践中，上述两种方案只选其一都太过极端，一般会结合两种复制方式使用：
    
    一般存在一个从节点是同步复制，其他从节点是异步复制，万一同步节点不可用，将提升另一个异步节点作为同步节点，这样就保证了一个集群中至少拥有两个数据一致且最新的节点
    

**新增节点**

场景：提高容错能力，替换故障节点等

原理简述：

1. 在某个时间点对主节点生成数据快照
2. 将此快照应用到新增节点
3. 新增节点连接至主节点，请求快照点（与某个日志顺序点关联）后的数据更改日志
    
    PostgreSQL将日志顺序点称为log sequence number，MySQL则称为binlog coordinates
    
4. 获取数据更改日志后，依次应用日志中的数据变更，这一步称为**追赶**

**处理节点失效**

- 从节点失效：追赶式恢复
    
    从节点崩溃下线后，又顺利重启，可以通过数据复制日志得知故障前处理的最后一笔事务，然后向主节点请求该事务后发生的所有数据变更日志，并应用到本地，从而**追赶主节点**
    
- 主节点失效：节点切换
    
    原理简述：
    
    1. 确认主节点失效。一般基于超时机制判断，节点间心跳包如果超过一段时间没有得到响应，则视为节点失效
    2. 选举新的主节点。从节点之间选举（超过半数节点达到共识），目标是选举出与主节点数据差异最小的一个从节点提升为主节点。这里涉及到共识算法，常见的有Raft等
    3. 重新配置系统激活主节点。主要就是客户端的数据写请求现在应该路由到新晋升的主节点。前主节点重新上线后还要确保其降级为从节点，并认可新的主节点
    
    存在的问题：
    
    1. 前主节点重新上线后，可能依旧认为自己是主节点，从而会继续尝试同步其他节点，导致新主节点产生写冲突
    2. 如果数据库需要和其他系统相协调，那么丢弃写入内容是极其危险的操作。比如说一个MySQL集群采用自增作为主键，如果一个数据未完全同步的MySQL从节点晋升为主节点，那么在后续插入行的操作下，新主节点将会重新分配旧主节点已经分配过的主键。倘若外部有一个redis引用了主键字段，就会发生MySQL与redis数据不一致的情况
    3. 某些故障下，可能会发生多个节点认为自己是主节点，这种现象称为**脑裂（split brain）**，这样就会导致多个节点可写，最后出现数据冲突的情况
    4. 节点失效的超时检测机制很难设置一个合适的时间，时间越长代表总体恢复时间越长，时间越短，越可能导致不必要的节点切换
    

**复制日志的实现**

- 基于语句的复制
    
    主节点记录每次写请求所执行的语句，并将语句日志发送给从节点。对于关系数据库，每个INSERT、UPDATE或DELETE语句都会转发给从节点，然后交由从节点执行。
    
    有下列几个问题：
    
    1. 任何非确定函数语句比如NOW()、RAND()，在不同节点上可能有不同返回值
    2. 如果语句使用了自增等依赖现有数据的情况，会受到限制
    3. 有副作用的语句，比如触发器、用户定义函数等，不同节点会产生不同的副作用
- 基于预写日志（WAL）传输
    
    Write-ahead logging，所有的修改在提交之前都要先写入log文件中，可以使用完全相同的日志文件复制一个内容和主节点完全相同的副本。
    
    有下列问题：
    
    日志记录的数据非常底层，WAL包含哪些磁盘块中的哪些字节发生了更改。这使复制与存储引擎紧密耦合。如果数据库将其存储格式从一个版本更改为另一个版本，通常不可能在主库和从库上运行不同版本的数据库软件。
    
- 基于行的逻辑日志复制
    
    关系数据库的逻辑日志通常是以行的粒度描述对数据库表的写入的记录序列：
    
    - 对于插入的行，日志包含所有列的新值。
    - 对于删除的行，日志包含足够的信息来唯一标识已删除的行。通常是主键，但是如果表上没有主键，则需要记录所有列的旧值。
    - 对于更新的行，日志包含足够的信息来唯一标识更新的行，以及所有列的新值（或至少所有已更改的列的新值）。
- 基于触发器的复制
    
    触发器支持注册自己的应用层代码，可以将数据更改记录记录到一个单独的表中，然后在应用层访问该表，并执行自定义逻辑，比如将数据复制到另一个系统
    

**复制滞后问题**

前面讨论过，实践中主从复制一般采用半同步方案，也就是存在一个同步节点和多个异步节点。而异步就意味着数据的非即时性。一个客户端同时查询主节点和一个异步从节点可能会得到不同的结果，这就是异步复制所带来的复制滞后问题。

尽管如果主节点停止写入一段时间，从节点会通过追赶过程，最终会达到与主节点数据一致（最终一致性），但是当滞后时间过长，将会发生如下几个问题：

- 读自己的写
    
    ![](ddia-3.png)
    
    用户在提交一些数据后再次读取提交的数据时，由于可能在一个数据尚未同步的从节点（Follower2）进行读取，会产生“数据似乎丢失”的假象。
    
    此时需要**read-after-write 一致性**（写后读一致性、读写一致性）来防止这种现象。
    
    通常有以下几种方案：
    
    1. 在主节点访问可能被用户自己修改的内容，否则就在从节点访问
    2. 跟踪最近更新时间，设定一个阈值m，用户更新后m分钟内，查询请求都将由主节点处理，否则在从节点处理；同时监控从节点，避免从复制滞后超过m分钟的节从节点读取。
    3. 客户端记录用户更新时间戳，并附带在请求中，服务端找到拥有此时间戳数据的节点后，就由此节点处理请求
- 单调读
    
    ![](ddia-4.png)
    
    如果用户从不同从库（一个新一个旧）进行多次读取，就可能发生这种情况
    
    **单调读一致性**可以避免此异常，此种一致性比强一致性弱，比最终一致性强，实现单调读一致性的一种实现方式是，利用用户唯一标识符进行hash，从而将用户分配到一个固定的节点上。
    
- 前缀一致读
    
    ![](ddia-5.png)
    
    如果某些分区的复制速度慢于其他分区，那么观察者在看到问题之前可能会看到答案。
    
    **前缀一致读（consistent prefix reads）**可以避免此种异常。
    
    这是分区（partitioned）（分片（sharded））数据库中的一个特殊问题，不同的分区独立运行，因此不存在全局写入顺序，一个解决方案是确保任何因果相关的写入都写入相同的分区，但是这就与分区的思想相违背了，在后续”Happens-before关系与并发小节“将介绍一个追踪事件因果关系的算法。
    

### 多主节点复制

每个主节点都可以接受写操作，并将写操作转发给其他节点。同时每个主节点扮演其他主节点的从节点。

**处理写冲突**

![](ddia-6.png)

两个用户的写操作都成功了，但是在后续主节点间同步时出现了写冲突

- 避免冲突
    
    应用层保证写请求总是被路由到同一个主节点，局限性高
    
- 收敛于一致状态
    
    在主从复制模型下，写请求是符合顺序性原则的，但在多主节点模型下不存在这种顺序性，通常有下列几种方式来收敛于一致状态：
    
    1. 给每个写入一个唯一的ID（例如，一个时间戳，一个长的随机数，一个UUID或者一个键和值的哈希），挑选最高ID的写入作为胜利者，并丢弃其他写入。如果使用时间戳，这种技术被称为最后写入胜利（LWW, last write wins）。虽然这种方法很流行，但是很容易造成数据丢失。
    2. 为每个副本分配一个唯一的ID，ID编号更高的写入具有更高的优先级。这种方法也意味着数据丢失。
    3. 以某种方式将这些值合并在一起 - 例如，按字母顺序排序，然后连接它们，在上图中，合并的标题可能类似于“B/C”）。
    4. 用一种可保留所有信息的显式数据结构来记录冲突，并编写解决冲突的应用程序代码（也许通过提示用户的方式）。
- 自定义冲突解决逻辑
    1. 在写入时执行
        
        只要数据库在复制变更日志时检测到了冲突，就调用应用层冲突处理程序
        
    2. 在读取时执行
        
        所有冲突值都会被记录下来，下一次读取数据时，将这些冲突值都返回给应用层，交由应用层处理
        

**拓扑结构**

![](ddia-7.png)

上图为多主节点模型的三种拓扑结构

在环形拓扑（a）和星形拓扑（b）中，写请求都要经过多个节点的传递才能到达所有节点。并且每个请求都附带了已通过的节点标识符，依此来避免请求传递的无线循环。在这两个拓扑类型中，如果有节点发生了故障，将导致整个系统复制日志的转发受到影响。而全部-全部拓扑模型（c）中，请求可以通过多种不同路径转发，避免了单点故障，提高了容错性。

这里有个请求到达顺序问题：

![](ddia-8.png)

由于网络阻塞等原因，对于同一个主键的update操作先于insert操作到达了主节点2了，这里为了使得消息正确有序，可以使用**版本矢量**技术，后文介绍。

### 无主节点复制

无主节点复制方案中，客户端将写请求直接发送到多个副本，标杆：亚马逊Dynamo系统

**节点失效时写入数据库**

在下图中

1. 一次写请求同时发送给了三个副本，其中副本三故障下线了，客户端收到两个节点的成功响应就认为写入成功
2. 故障节点重新上线后，为了避免读取到旧数据，客户端同样并发地向多个节点发送读请求
3. 客户端可能读到不同的结果，为了确定哪个是最新值，引入版本号技术

![](ddia-9.png)

**读修复和反熵**

在上图最后，客户端将读取到的最新值有重新写回了落后副本，这个方案称为**读修复**，主要适合频繁读取的场景。

此外还有一种机制被经常应用于无主节点系统的**追赶**过程，**反熵。**一些数据存储具有不断查找副本之间的数据差异的后台进程，并将任何缺少的数据从一个副本复制到另一个副本。与主从复制中的复制日志不同，此反熵过程不会以任何特定的顺序复制写入，并且在复制数据之前可能会有显著的延迟。

**读写quorum**

假设有n个副本，写入需要w个节点确认，读取至少需要通过r个节点查询，那么n,w,r三者的关系应该是怎样的？可以很简单的得到`n < w + r` ，只有这样读请求才能保证至少有一个结果是最新值。满足上述条件的w,r我们称之为**仲裁写，仲裁读。**通常，读取和写入操作总是并行地发送给全部n个节点，w和r只是决定需要返回结果的节点数。

- 读多写少：可以设置w=n，r=1
- 读少写多：可以设置w=1，r=n

当然上述两种方案都是最极端的设置，需要承担单点故障的风险，可以根据情况适当调整w，r值

**quorum一致性的局限性**

一般情况下，设定w+r>n可以保证至少有一个最新值被读取到

但是我们也可以设置w+r≤n，虽然读取请求最终可能返回一个旧值，但是这样的配置能够获得更低的延迟和更高的可用性，比较适合对数据实时性要求没那么高的场景

需要注意的是，即使设定w+r>n， 也可能存在返回旧值的边界条件：

1. 如果采用了sloppy quorum（后面会讲），w个写入和r个读取可能落在完全不同的节点上，就无法保证读到最新值
2. 如果两个写入同时发生，不清楚哪一个先发生。在这种情况下，唯一安全的解决方案是合并并发写入（处理写入冲突）。如果根据时间戳（最后写入胜利）挑选出一个胜者，则由于时钟偏差写入可能会丢失。
3. 如果写操作与读操作同时发生，写操作可能仅反映在某些副本上。在这种情况下，不确定读取是返回旧值还是新值。
4. 如果写操作在某些副本上成功，而在其他节点上失败（例如，因为某些节点上的磁盘已满），且成功的数量小于w，所以系统判定写入失败，但是写入成功的副本并不会回滚写入的数据。这意味着尽管写入失败，后续的读取仍然可能会读取这次失败写入的值。
5. 如果具有新值的节点发生失效，恢复重启时，恢复数据来自旧值，则具有新值的副本数将小于w，打破了quorum条件
6. 即使一切工作正常，有时也会出现一些边界情况，书中第九章可线性化与quorum有介绍

这里依旧存在前文“复制滞后问题”中所列举出的一系列问题，如果确实需要更强的保证，就需要**事务**与**共识**的问题了。

**宽松的quorum（sloppy quorum）与数据回传**

写和读仍然需要w和r成功的响应，但是可能包括不在指定的n个节点中的临时节点。

详细点来说就是当客户端向指定的n个节点中写入时，无法得到w个节点的响应，那么系统允许将失败的那些写请求写入到临时节点，并且还是判定写成功。等到节点恢复后，临时节点就会将暂存的值写回到原始节点上，这个过程叫做**数据回传**。

由此可见，在宽松的quorum下，即使满足w+r>n，也不能保证能够一定读取到新值。但是它提高了很大的写入可用性。

所有Dynamo风格的系统都已经支持了sloppy quorum。

**检测并发写**

与多主节点复制类似（见前文【处理写冲突】），无主节点复制在并发对某个主键进行写时，也会出现写冲突，此外读修复和数据回传同样会导致并发写冲突。

![](ddia-10.png)

- 节点 1 接收来自 A 的写入，但由于暂时中断，没接收到来自 B 的写入。
- 节点 2 首先接收来自 A 的写入，然后接收来自 B 的写入。
- 节点 3 首先接收来自 B 的写入，然后接收来自 A 写入。

如果像上面过程一样，每当接收到一个写入请求就简单的覆盖掉原来的值，那么整个系统将达不过到一致状态。

在前文多主节点处理写冲突小节，有说到“收敛于一致性状态”，其中一个方案就是每个副本总是保存最新值。如果以客户端时间戳来定义最新，我们可以称其为：

**最后写入者获胜（last write wins，LWW）**

LWW可以实现最终收敛，但是牺牲了一定的数据持久性，另外由于分布式系统中**不可靠时钟**的挑战，LWW甚至会删除非并发写。

对于缓存系统，数据持久性要求不高的场景LWW倒是挺适合的。

**Happens-before关系和并发**

如果一个操作发生在另一个操作之前，则后面的操作可以覆盖之前的操作，属于因果关系。如果两个操作都不在另一个之前发生或者都不知道对方的发生，那么就属于并发关系，这就需要解决可能发生的写入冲突。

**确定前后关系**

![](ddia-11.png)

- 服务器为每个键保留一个版本号，每次写入键时都增加版本号，并将新版本号与写入的值一起存储。
- 当客户端读取键时，服务器将返回所有未覆盖的值以及最新的版本号。客户端在写入前必须先发送读取请求。
- 客户端写入键时，必须包含之前读取的版本号，并且必须将之前读取的所有值合并在一起。来自写入请求的响应可以像读取一样，返回所有当前值。
- 当服务器接收到具有特定版本号的写入时，它可以覆盖该版本号或更低版本的所有值（因为它知道它们已经被合并到新的值中），但是它必须保留更高版本号的值（因为这些值与当前的写操作属于并发）。

**版本矢量**

上述过程是只有一个副本的情况，在此副本服务端引入一个版本号技术就能够确定多个写请求是属于因果关系还是并发关系。

如果扩展到无主节点集群，由于每个副本都能够接受写请求，那么单个的版本号就无法满足要求了，这就要求每个副本和每个主键都要维护一个版本号，并且同时还要跟踪其他副本的版本号，通过这些信息才能判断写请求的依赖关系。

所有副本的版本号集合就被成为是**版本矢量。**