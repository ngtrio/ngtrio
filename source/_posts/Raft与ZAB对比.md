---
title: Raft与ZAB对比
tags: 
- 分布式
---

## **ZAB**

### **ZAB节点状态：**

1. LOOKING
2. FOLLOWING
3. LEADING
4. OBSERVING

### **专有名词**

1. electionEpoch：选举的逻辑时钟
2. peerEpoch：每次leader选举完成后会选出一个peerEpoch
3. zxid：每个proposal的唯一id，高32位为peerEpoch低32位为counter
4. lastProcessedZxid：最后一次commit的zxid

### **理论实现的四个阶段**

**Phase 0. Leader election**

所有节点最开始都是LOOKING。只要有一个节点得到超半数节点的票数，它就可以当选准 leader。只有到达 Phase 3 准 leader 才会成为真正的 leader。这一阶段的目的是就是为了选出一个准 leader，然后进入下一个阶段。

协议并没有规定详细的选举算法。

**Phase 1. Discovery**

这个阶段有两个工作

1. 获取所有follower的lastZxid确定当前集群中有哪个节点拥有最新数据
2. 从所有follower的currentEpoch中选出一个最大的然后自增1得到peerEpoch，并发给所有follower，follower会将自己的acceptEpoch设置为peerEpoch，拒绝一切小于该epoch的请求

**Phase 2. Synchronization**

这个阶段就是根据Discovery阶段找到的最新数据节点，leader会与其同步。

**这里发生了follower到leader的数据同步，这是和zookeeper的实现还有raft的实现是不一样的**

同步完成后，会向所有follower同步数据，只有当quorum的follower都完成了数据同步后，其当选为新的leader。

**Phase 3 . Broadcast**

到了这个阶段， leader才能对外提供服务，可以进行消息广播。数据同步的过程类似一个2PC，Leader将client发过来的请求生成一个事务proposal，然后发送给Follower，多数Follower应答之后，Leader再发送Commit给全部的Follower让其进行提交。

### **Zookeeper的实现**

**Phase 0. Fast Leader Election**

这个阶段相当于是理论实现中的Phase 0，Phase 1的整合。每个节点不断更新自己的票箱，最终能够找到lastZxid最大的节点，并将其推选为leader。

这样的实现也避免了sync的时候需要从follower向leader同步数据。

成为leader的条件

- epoch最大
- zxid最大
- server id最大

节点在选举开始都默认投票给自己，当接收其他节点的选票时，会根据上面的条件更改自己的选票并重新发送选票给其他节点，当有一个节点的得票超过半数，该节点会设置自己的状态为 leading，其他节点会设置自己的状态为 following。

**Phase 1. Recovery**

这个阶段所有的follower都会发送自己的lastZxid到leader。

Leader会根据follower的lastZxid和自己的lastZxid进行比较，做出如下三种可能的同步策略：

1. SNAP：如果follower数据太老，已经小于minCommitLog则采取快照同步
2. DIFF：如果follower的lastZxid 处于minCommitLog和maxCommitLog之间，则采取增量同步F.lastZxid-L.lastZxid之间的数据
3. TRUNC：当F.lastZxid比L.lastZxid大时，Leader会让follower删除所有对于的数据

**Phase 2. Broadcast**

同理论实现

## **Raft**

### **Raft节点状态：**

1. FOLLOWER
2. CANDIDATE
3. LEADER

**触发Leader选举时机**

1. 当整个集群初始化的时候，所有节点都是Follower，此时等到超时的节点会转变为Candidate发起RequestVoteRPC发起选举
2. 当leader down掉后，由于不再有AppendEntriesRPC来维持心跳，follower也会发生超时，开始选举

**Leader Election**

当Follower/Candidate发生超时

1. 首先将自己的Term自增1
2. 然后投票给自己
3. 会向集群中的所有节点发送RequestVoteRPC

当收到majority的选票后，自己会转变为Leader，此时会开始向所有其他节点发送AppendEntriesRPC。

Follower投票的条件：

1. Candidate的Term比自身的Term大
2. 如果Term一样，那么就需要candidate的last log index比自己的大

通过上述条件选出来的leader能够保证会拥有所有已经提交的entry

**Log Replication**

选举完成后：

1. 当一个节点成为leader后会初始化两个数组
    1. nextIndex，记录所有follower的下一个日志index，初始值为leader的last log index
    2. matchIndex，记录所有follower已经确定完成同步的日志index
2. 然后通过AppendEntriesRPC完成数据同步：
    1. prevLogIndex：记录上一条日志index
    2. prevLogTerm：记录上一条日志Term
3. 只要follower发现没有一个下标为prevLogIndex并且term为prevLogTerm的entry都会返回false，leader便会把其对应的nextIndex减一，一直重复这个过程直到两者match，并更新matchIndex
4. 后续AppendEntriesRPC便会逐渐将Follower的日志补齐

处理客户端命令：

1. Leader处理读，每个entry只有append到大多数节点的时候，主节点才视为commit，并且更新commitIdex，随后这个commitIndex会被AppendEntriesRPC逐渐同步到Follower
2. 所有节点，只要发现commitIndex>lastApplied都会将entry提交到状态机

**Log Compaction**

由于内存是有限的，所有的节点都会定期对缓存的log进行compact生成snapshot，对于远远落后的follower，主节点会发送InstallSnapshotRPC给follower,这个过程是分成一个个chunk来完成的

### **两者不同点**

1. 触发选举方式不同
    - raft follower超时+prevote避免term无限递增
    - zab follower超时+leader发现大多数follower超时
2. 选举机制不同
    - raft每个term只会投一票，存在split vote可能性
    - zab每个epoch会不断更新选票，时间理论上相对Raft要花费的多。
3. 日志同步机制不同，并且对未提交日志的处理方式不同
    - Raft中leader是根据AppendEntriesRPC的nextIndex，prevLogIndex，prevLogTerm来跟踪follower的同步情况，并实现逐步的同步，期间有个election restriction，限制term只能commit属于term的entry，旧term的entry只能被当前term的commit给附带commit。所以raft中未提交的日志可能提交也可能会被覆盖
    - ZAB在选主后，有一个Recovery阶段，根据每个节点的lastZxid来判断日志的取舍，在leader上所有没有commit的日志都会提交。
4. 残留日志处理
    - Raft：对于之前term的过半或未过半复制的日志采取的是保守的策略，全部判定为未提交，只有当当前term的日志过半了，才会顺便将之前term的日志进行提交
    - ZooKeeper：采取激进的策略，对于所有过半还是未过半的日志都判定为提交，都将其应用到状态机中