# 02-Schedule Framework

## Scheduling Framework 调度框架

_scheduling framework_  是 Kubernetes 调度器的可插拔架构，它由一组直接编译到调度程序中的“plugin”API 组成。

## Framework workflow 框架工作流程

调度框架定义了一些扩展点，调度程序插件注册以在一个或多个扩展点处调用。

每次尝试调度一个 Pod 都会分为两个阶段：**scheduling cycle（**&#x8C03;度周期）和**binding cycle（**&#x7ED1;定周期）。

## 调度周期&绑定周期

调度周期为 Pod 选择一个节点，绑定周期将该决策提交给集群。调度周期和绑定周期一起被称为 “scheduling context”（调度上下文）。

调度周期串行运行，而绑定周期可以并发运行。

如果 Pod 被确定为不可调度或存在内部错误，则可以中止调度或绑定周期。 Pod 将返回队列并重试

### Interfaces <a href="#interfaces" id="interfaces"></a>

下图展示了Pod的调度上下文以及调度框架暴露的接口。

<figure><img src="../../../.gitbook/assets/image (6) (1) (1).png" alt=""><figcaption></figcaption></figure>

### PreEnqueue

这些插件在将 Pod 添加到内部 active queue (活动队列)之前被调用，其中 Pod 被标记为准备调度。

只有当所有 PreEnqueue 插件都返回 `Success` 时，Pod 才被允许进入活动队列。否则，它将被放置在内部不可调度的 Pod 列表中，并且不会获得 `Unschedulable` 条件。

排队机制是调度程序的一个组成部分。它允许调度程序为下一个调度周期选择最合适的 Pod。鉴于 pod 可以指定调度时必须满足的各种条件，例如持久卷的存在、遵守 pod 反亲和性规则或节点污点的容忍度，该机制需要能够推迟调度操作直到集群满足调度成功的所有条件。该机制依赖于三个队列：

* activeQ：提供pod进行即时调度
* unschedulableQ：用于等待某些条件发生的pod
* podBackoffQ：以指数方式推迟未能调度的 pod（例如仍在创建卷）但预计最终会得到调度。

activeQ 内部实现是一个 `Heap` 堆结构，按照优先级从高到低排序，堆顶的 Pod 优先级最高，新创建的 Pod (.spec.nodeName 属性为空) 都会加入到这个队列中。

在每个调度周期中，调度器会从该队列中取出 Pod 并进行调度，如果调度失败了 (例如因为资源不足)，Pod 就会被加入到 UnSchedulable 队列中。如果 Pod 被成功调度，**其会从队列中被删除**。

unschedulablePods **存放等待重试的 Pod**，之所以单独设置一个队列，是为了避免处于等待状态的 Pod 不断重试，给调度器带来不必要的负载。

内部实现是一个 `Heap` 堆结构，按照重试等待时间从低到高排序，堆顶的 Pod 等待时间最少。单个 Pod 的的重试次数越多，该 Pod 重新进入 Active 队列所需要的时间就越长 (几乎所有重试机制都是这么设计的)。

重试算法采用的是 **指数退避机制**，默认情况下最小为 1 秒，最大为 10 秒，例如，重试 3 次的 Pod 下一次的重试等待时间为 2^3 = 8 秒。为了避免 Pod 经常调度失败而频繁进入等待队列，应该配置合理的退避时间基数，降低系统负载。

此外，调度队列机制有两个在后台运行的 goroutine, 负责定期刷新 将 Pod 加入到 Active 队列，当某些事件 (例如节点添加或更新、现有 Pod 被删除等) 触发时， 调度程序会将 UnSchedulable 队列或 Backoff 队列中的 Pod 移动到 Active 队列，做好重新调度前的准备工作。

* flushUnschedulableQLeftover：每 30 秒运行一次，将 Pod 从不可调度队列中移动，以允许未由任何事件移动的不可调度 Pod 再次重试。 Pod 必须在队列中停留至少 30 秒才能移动。在最坏的情况下，最多可能需要 60 秒才能移动 Pod。
* flushBackoffQCompleted：运行回退到活动队列足够长时间的第二个移动 Pod。

<figure><img src="../../../.gitbook/assets/image (7) (1).png" alt=""><figcaption></figcaption></figure>

```go
// PriorityQueue implements a scheduling queue.
// The head of PriorityQueue is the highest priority pending pod. This structure
// has three sub queues. One sub-queue holds pods that are being considered for
// scheduling. This is called activeQ and is a Heap. Another queue holds
// pods that are already tried and are determined to be unschedulable. The latter
// is called unschedulableQ. The third queue holds pods that are moved from
// unschedulable queues and will be moved to active queue when backoff are completed.
type PriorityQueue struct {
	// PodNominator abstracts the operations to maintain nominated Pods.
	framework.PodNominator

	stop  chan struct{}
	clock util.Clock

	// pod initial backoff duration.
	podInitialBackoffDuration time.Duration
	// pod maximum backoff duration.
	podMaxBackoffDuration time.Duration

	lock sync.RWMutex
	cond sync.Cond

	// activeQ is heap structure that scheduler actively looks at to find pods to
	// schedule. Head of heap is the highest priority pod.
	activeQ *heap.Heap
	// podBackoffQ is a heap ordered by backoff expiry. Pods which have completed backoff
	// are popped from this heap before the scheduler looks at activeQ
	podBackoffQ *heap.Heap
	// unschedulableQ holds pods that have been tried and determined unschedulable.
	unschedulableQ *UnschedulablePodsMap
	// schedulingCycle represents sequence number of scheduling cycle and is incremented
	// when a pod is popped.
	schedulingCycle int64
	// moveRequestCycle caches the sequence number of scheduling cycle when we
	// received a move request. Unschedulable pods in and before this scheduling
	// cycle will be put back to activeQueue if we were trying to schedule them
	// when we received move request.
	moveRequestCycle int64

	// closed indicates that the queue is closed.
	// It is mainly used to let Pop() exit its control loop while waiting for an item.
	closed bool
}

// Run starts the goroutine to pump from podBackoffQ to activeQ
func (p *PriorityQueue) Run(logger klog.Logger) {
	go wait.Until(func() {
		p.flushBackoffQCompleted(logger)
	}, 1.0*time.Second, p.stop)
	go wait.Until(func() {
		p.flushUnschedulablePodsLeftover(logger)
	}, 30*time.Second, p.stop)
}
```

### Scheduling context

1.`queueSort` ：这些插件提供排序功能，用于对调度队列中待处理的 Pod 进行排序。一次只能启用一个队列排序插件。

`2.preFilter` ：这些插件用于在过滤之前预处理或检查有关 Pod 或集群的信息。他们可以将 Pod 标记为unschedulable（不可调度）。

3.`filter` ：原来的预选阶段（Predicate），用于过滤掉不能运行Pod的节点。过滤插件按照配置的顺序调用。如果没有节点通过所有过滤插件，则 Pod 被标记为不可调度。

4.`postFilter` ：当 pod 没有找到可行的节点时，这些插件将按照其配置顺序调用。如果任何 `postFilter` 插件将 Pod 标记为可调度，则不会调用其余插件。

5.`preScore` ：这是一个信息扩展点，可用于进行预评分工作。

6.`score` ：这些插件为每个通过 filter 阶段的节点提打分。然后调度程序将选择加权分数总和最高的节点。

7.`reserve` ：这是一个信息扩展点，用于在为给定 Pod 保留资源时通知插件。插件还实现 `Unreserve` 调用，如果在 `Reserve` 期间或之后发生故障，则会调用该调用。

8.`permit` ：这些插件可以阻止或延迟 Pod 的绑定。

9.`preBind` ：这些插件在绑定 Pod 之前执行所需的任何工作。

10.`bind` ：插件将 Pod 绑定到 Node。 `bind` 插件按顺序调用，一旦完成绑定，其余插件将被跳过。至少需要一个绑定插件。

11.`postBind` ：这是一个信息扩展点，在 Pod 绑定后调用。



对于每个扩展点，您可以禁用特定的默认插件或启用您自己的插件。例如：

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
profiles:
  - plugins:
      score:
        disabled:
        - name: PodTopologySpread
        enabled:
        - name: MyCustomPluginA
          weight: 2
        - name: MyCustomPluginB
          weight: 1
```

整体路程如下图：

<figure><img src="../../../.gitbook/assets/image (8).png" alt=""><figcaption></figcaption></figure>
