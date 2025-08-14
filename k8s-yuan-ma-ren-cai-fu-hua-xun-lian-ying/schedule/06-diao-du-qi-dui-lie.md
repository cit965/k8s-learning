# 06-调度器队列

## 概述

**调度器队列** 在 k8s 调度中起着非常重要的作用，在阅读本文前不如思考如下问题:

* 当有一批 Pod 同时需要调度时，它们被调度的先后顺序是怎样的？
* 当某些 Pod 调度失败时，会被存储到哪里？
* 对于调度失败的 Pod, 它们的重试机制是怎样的？

为了解答上面的这些问题，就需要深入了解 **调度器队列** 的实现机制。

## 为什么需要调度队列？

* 当 Pod 要求特定条件（如：“需要 SSD 存储卷”或“不能和某类 Pod 同节点”）时，可能需等待条件满足
* 避免频繁重试失败任务浪费资源
* 实现优先级调度和故障恢复

## 三大核心队列：调度器的“候诊室”

scheduler 通过 watch 机制监听 pod.Spec.NodeName 为空的pod，并把pod 加入优先级队列等待调度。优先级队列又由三个子队列组成，分别是 activeQ,BackoffQ,unschedulableQ (如下图):

<figure><img src="../../.gitbook/assets/截屏2025-08-14 17.53.07.png" alt=""><figcaption></figcaption></figure>

### 1. 急诊室 🚑 `activeQ`（活动队列）

* **作用**：立即准备调度的 Pod 队列
* **特点**：
  * 默认按**优先级排序**（可自定义排序规则）
  * 新创建的 Pod 直接进入此队列
  * 调度器每次从此队列取一个 Pod 进行调度
* **行为类比**：就像医院的急诊室，随时处理最紧急的病人

### 2. 观察室 🕒 `unschedulableQ`（不可调度队列）

* **作用**：存放**暂时无法调度**的 Pod（如等待节点资源释放）
* **特点**：
  * 使用 Map 结构快速查找
  * 等待外部事件触发（例如新节点加入、存储卷创建完成）
* **行为类比**：留观病房里的患者，需等待检查结果出来才能治疗

### 3. 复查等待区 ⏳ `podBackoffQ`（退避队列）

* **作用**：存放**需要延迟重试**的调度失败 Pod
* **核心机制**：**指数退避策略**
  * 初始等待：1 秒（默认）
  * 最大等待：10 秒（默认）
  * 计算公式：`等待时间 = min(初始值 × 2^失败次数, 最大值)`
  * 示例：
    * 第 1 次失败 → 等待 2 秒 (1×2¹)
    * 第 3 次失败 → 等待 8 秒 (1×2³)
    * 第 5 次失败 → 等待 32 秒 → **但被限制为 10 秒**
* **行为类比**：复查患者需间隔一段时间再检查，失败次数越多等待越久

## 源码分析

我们主要看下优先级队列的2个接口,想深入了解代码的话可以直接看k8s源码：

`PriorityQueue.Add` 方法负责将 Pod 添加到 activeQ 队列中。

```go
func (p *PriorityQueue) Add(pod *v1.Pod) error {
 pInfo := p.newQueuedPodInfo(pod)
 // 添加到 activeQ 队列
 if err := p.activeQ.Add(pInfo); err != nil {
  return err
 }
 // 从 unschedulableQ 队列中删除该 Pod
 if p.unschedulableQ.get(pod) != nil {
  p.unschedulableQ.delete(pod)
 }
 // 从 backoffQ 队列中删除该 Pod
 if err := p.podBackoffQ.Delete(pInfo); err == nil {
 }

 ...

 return nil
}
```

`PriorityQueue.Pop` 方法负责将 activeQ 队列的堆顶元素取出并返回。

```go
func (p *PriorityQueue) Pop() (*framework.QueuedPodInfo, error) {
 // 如果 activeQ 队列为空
 // 那么这里就变为阻塞操作
 for p.activeQ.Len() == 0 {
  if p.closed {
   return nil, fmt.Errorf(queueClosed)
  }
  // 等待通知
  p.cond.Wait()
 }

 // 弹出堆顶元素并返回
 obj, err := p.activeQ.Pop()
 pInfo := obj.(*framework.QueuedPodInfo)

 ...

 return pInfo, err
}
```

## 总结

通过三个队列的相互配合，调度器队列实现了调度过程中 Pod 获取、 Pod 状态变化、Pod 存储等重要功能，在分析源代码的过程中，我们又遇到了 `队列`、`堆数据结构`、`指数退避算法` 等基础知识， 即使如 Kubernetes 这般庞然大物，其内部也就是由基础知识一点一滴搭建起来的，学习优秀的开源项目本身也是个复习巩固的过程，同时也希望读者能常读常新。
