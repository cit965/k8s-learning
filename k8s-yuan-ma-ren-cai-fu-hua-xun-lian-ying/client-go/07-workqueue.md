# 07-workqueue

<figure><img src="../../.gitbook/assets/image (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

今天我们来详细研究下 workqueue 相关代码。client-go 的 util/workqueue 包里主要有三个队列，分别是普通队列，延时队列，限速队列，后一个队列以前一个队列的实现为基础，层层添加新功能，我们按照 Queue、DelayingQueue、RateLimitingQueue 的顺序层层拨开来看限速队列是如何实现的。



## &#x20;queue

#### 接口和结构体：

k8s.io/client-go/util/workqueue/queue.go

```go
type TypedInterface[T comparable] interface {
	Add(item T)
	Len() int
	Get() (item T, shutdown bool)
	Done(item T)
	ShutDown()
	ShutDownWithDrain()
	ShuttingDown() bool
}
```

这个队列的接口定义很清楚，我们来看其实现的类型：

```go
type Typed[t comparable] struct {
	// queue defines the order in which we will work on items. Every
	// element of queue should be in the dirty set and not in the
	// processing set.
	// 定义元素的处理顺序，里面所有元素都应该在 dirty set 中有，而不能出现在 processing set 中
	queue Queue[t]

	// dirty defines all of the items that need to be processed.
	// 标记所有需要被处理的元素
	dirty set[t]

	// Things that are currently being processed are in the processing set.
	// These things may be simultaneously in the dirty set. When we finish
	// processing something and remove it from this set, we'll check if
	// it's in the dirty set, and if so, add it to the queue.
	// 存放正在被处理的元素，可能同时存在于dirty set。 当我们完成处理后，会将其删除，我们会看看他是否在 dirty set 中，如果在，添加到 queue中
	processing set[t]

	// 条件变量，在多个goroutines等待、1个goroutine通知事件发生时使用
	cond *sync.Cond

	shuttingDown bool
	drain        bool

	metrics queueMetrics

	unfinishedWorkUpdatePeriod time.Duration
	clock                      clock.WithTicker
}

// queue is a slice which implements Queue.
type queue[T comparable] []T
```

### Add

```go
// Add marks item as needing processing.
// 将元素标记成需要处理
func (q *Typed[T]) Add(item T) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	// 队列正关闭，则直接返回
	if q.shuttingDown {
		return
	}
	// 已经标记为 dirty 的数据，也直接返回，因为存储在了脏数据的集合中
	if q.dirty.has(item) {
		// the same item is added again before it is processed, call the Touch
		// function if the queue cares about it (for e.g, reset its priority)
		// 相同的元素又被添加进来了，如果queue 关心他，调用 touch,这里需要自己去实现 queue的 touch方法，自带的不会处理
		if !q.processing.has(item) {
			q.queue.Touch(item)
		}
		return
	}

	q.metrics.add(item)

	// 添加到脏数据集合中
	q.dirty.insert(item)
	// 元素如果正在被处理，那就直接返回
	if q.processing.has(item) {
		return
	}
	// 追加到元素数组的尾部
	q.queue.Push(item)
	// 通知有新元素到了，此时有协程阻塞就会被唤醒
	q.cond.Signal()
}
```

问题：

为啥在添加数据的同时要添加到 dirty 脏数据集合中呢，存储在 queue 中不就可以了么？

* 为了让 queue 中不存在重复的 items，所以加了一个 dirty set，毕竟判断 map 中是否存在某个 key 比判断 slice 中是否存在某个 item 要快得多。
* 队列中曾经存储过该元素，但是已经被拿走还没有调用 `Done()` 方法时，也就是正在处理中的元素，此时再添加当前的元素应该是最新的，处理中的应该是过时的，也就是脏的。

### Get

`Get()` 方法尝试从 `queue` 中获取第一个 item，同时将其加入到 `processing set` 中，并且从 `dirty set`中删除。

```go
 func (q *Type) Get() (item interface{}, shutdown bool) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	// 如果当前队列中没有数据，并且没有要关闭的状态则阻塞协程
	for len(q.queue) == 0 && !q.shuttingDown {
		q.cond.Wait()
	}
	// 协程被激活但还没有数据，说明队列被关闭了
	if len(q.queue) == 0 {
		// We must be shutting down.
		return nil, true
	}
	// 从队列中弹出第一个元素
	item = q.queue[0]
	// The underlying array still exists and reference this object, so the object will not be garbage collected.
	q.queue[0] = nil
	q.queue = q.queue[1:]

	q.metrics.get(item)
	// 加入到处理队列中
	q.processing.insert(item)
	 // 同时从dirty集合（需要处理的元素集合）中移除
	q.dirty.delete(item)

	return item, false
}
```

### Done

`Done()` 方法用来标记一个 item 被处理完成了。调用 `Done()` 方法的时候，这个 item 被从 `processing set` 中删除。

```go
 func (q *Type) Done(item interface{}) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	q.metrics.done(item)
 	 // 从正在处理的集合中删除元素
	q.processing.delete(item)
	 // 此处判断脏数据集合，如果在处理期间又被添加回去了，则又放到队列中重新处理。
	if q.dirty.has(item) {
		q.queue = append(q.queue, item)
		q.cond.Signal()
	} else if q.processing.len() == 0 {
		q.cond.Signal()
	}
}
```

## 延迟队列

```go
// TypedDelayingInterface is an Interface that can Add an item at a later time. This makes it easier to
// requeue items after failures without ending up in a hot-loop.
// DelayingInterface 是一个可以在一个时间后添加一个元素的 Interface。这使得在失败后重新排列元素更容易而不会在热循环中结束。
type TypedDelayingInterface[T comparable] interface {
	TypedInterface[T]
	// AddAfter adds an item to the workqueue after the indicated duration has passed
	AddAfter(item T, duration time.Duration)
}

```

从上面的定义来看延时队列和之前的通用队列基本上一致，只是多了延迟添加的接口，所以相当于在之前的通用队列基础上会添加一些机制来实现延迟添加，如下类型定义所示：

```go
// delayingType wraps an Interface and provides delayed re-enquing
type delayingType[T comparable] struct {
	TypedInterface[T]

	// clock tracks time for delayed firing
	clock clock.Clock

	// stopCh lets us signal a shutdown to the waiting loop
	stopCh chan struct{}
	// stopOnce guarantees we only signal shutdown a single time
	stopOnce sync.Once

	// heartbeat ensures we wait no more than maxWait before firing
	heartbeat clock.Ticker

	// waitingForAddCh is a buffered channel that feeds waitingForAdd
	// 所有延迟添加的元素封装成 waitFor 放到缓冲队列中
	waitingForAddCh chan *waitFor

	// metrics counts the number of retries
	metrics retryMetrics
}
// waitFor holds the data to add and the time it should be added
type waitFor struct {
	data    t
	readyAt time.Time
	// index in the priority queue (heap)
	index int
}

```

在这个基础上还定义了一个 `waitForPriorityQueue`，用来实现 `waitFor` 元素的优先级队列，把需要延迟的元素形成了一个队列，按照元素的延时添加的时间（readyAt）从小到大排序。



这里我们只需要知道 `waitForPriorityQueue` 是一个有序的 slice，排序方式是按照时间从小到大排序的，根据 `heap.Interface` 的定义，我们需要实现 `Len`、`Less`、`Swap`、`Push`、`Pop` 这几个方法：

```go
// waitForPriorityQueue implements a priority queue for waitFor items.
//
// waitForPriorityQueue implements heap.Interface. The item occurring next in
// time (i.e., the item with the smallest readyAt) is at the root (index 0).
// Peek returns this minimum item at index 0. Pop returns the minimum item after
// it has been removed from the queue and placed at index Len()-1 by
// container/heap. Push adds an item at index Len(), and container/heap
// percolates it into the correct location.
type waitForPriorityQueue []*waitFor

func (pq waitForPriorityQueue) Len() int {
	return len(pq)
}
func (pq waitForPriorityQueue) Less(i, j int) bool {
	return pq[i].readyAt.Before(pq[j].readyAt)
}
func (pq waitForPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push adds an item to the queue. Push should not be called directly; instead,
// use `heap.Push`.
func (pq *waitForPriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*waitFor)
	item.index = n
	*pq = append(*pq, item)
}

// Pop removes an item from the queue. Pop should not be called directly;
// instead, use `heap.Pop`.
func (pq *waitForPriorityQueue) Pop() interface{} {
	n := len(*pq)
	item := (*pq)[n-1]
	item.index = -1
	*pq = (*pq)[0:(n - 1)]
	return item
}
```

因为延时队列利用 `waitForPriorityQueue` 队列管理所有延时添加的元素，所有的元素在 `waitForPriorityQueue` 中按照时间从效到大排序，这样延时队列的处理就会方便很多了。



下面我们来看下延时队列的实现，由于延时队列包装了通用队列，所以我们只需要查看新增的实现延时的函数即可：

```go
// AddAfter adds the given item to the work queue after the given delay
func (q *delayingType[T]) AddAfter(item T, duration time.Duration) {
	// don't add if we're already shutting down
	if q.ShuttingDown() {
		return
	}

	q.metrics.retry()

	// immediately add things with no delay
	if duration <= 0 {
		q.Add(item)
		return
	}

	select {
	case <-q.stopCh:
		// unblock if ShutDown() is called
	case q.waitingForAddCh <- &waitFor{data: item, readyAt: q.clock.Now().Add(duration)}:
	}
}
```

`AddAfter()` 就是简单把元素送到 channel 中，所以核心实现是从 channel 中获取数据的部分，如下所示：

```go
// waitingLoop runs until the workqueue is shutdown and keeps a check on the list of items to be added.
func (q *delayingType[T]) waitingLoop() {
	defer utilruntime.HandleCrash()

	// Make a placeholder channel to use when there are no items in our list
	never := make(<-chan time.Time)

	// Make a timer that expires when the item at the head of the waiting queue is ready
	var nextReadyAtTimer clock.Timer
 	// 初始化上面的有序队列
	waitingForQueue := &waitForPriorityQueue{}
	heap.Init(waitingForQueue)
  	 // 这个map用来避免重复添加，如果重复添加则只更新时间即可
	waitingEntryByData := map[t]*waitFor{}

	for {
	 	// 队列关闭了则直接返回
		if q.TypedInterface.ShuttingDown() {
			return
		}

		now := q.clock.Now()

		// Add ready entries
		// 判断有序队列中是否有元素
		for waitingForQueue.Len() > 0 {
		  	// 获得有序队列中的第一个元素
			entry := waitingForQueue.Peek().(*waitFor)
			 // 元素指定的时间是否过了？没有的话就跳出循环
			if entry.readyAt.After(now) {
				break
			}
			 // 如果时间已经过了，那就从有序队列中拿出来放入通用队列中
			entry = heap.Pop(waitingForQueue).(*waitFor)
			q.Add(entry.data.(T))
			delete(waitingEntryByData, entry.data)
		}

		// Set up a wait for the first item's readyAt (if one exists)
		nextReadyAt := never
		 // 如果有序队列中有元素，那就用第一个元素指定的时间减去当前时间作为等待时间
		if waitingForQueue.Len() > 0 {
			if nextReadyAtTimer != nil {
				nextReadyAtTimer.Stop()
			}
			entry := waitingForQueue.Peek().(*waitFor)
			nextReadyAtTimer = q.clock.NewTimer(entry.readyAt.Sub(now))
			nextReadyAt = nextReadyAtTimer.C()
		}
		// 进入各种等待
		select {
		case <-q.stopCh:
			return

		case <-q.heartbeat.C():
			// continue the loop, which will add ready items
		 // 这个就是有序队列里面需要等待时间的信号，时间到就会有信号
		case <-nextReadyAt:
			// continue the loop, which will add ready items
		// 这里是从channel中获取元素，AddAfter()放入到channel中的元素
		case waitEntry := <-q.waitingForAddCh:
			// 时间没有过就插入到有序队列中
			if waitEntry.readyAt.After(q.clock.Now()) {
				insert(waitingForQueue, waitingEntryByData, waitEntry)
			} else {
				// 如果时间已经过了就直接放入通用队列
				q.Add(waitEntry.data.(T))
			}

			drained := false
			for !drained {
				select {
				case waitEntry := <-q.waitingForAddCh:
					if waitEntry.readyAt.After(q.clock.Now()) {
						insert(waitingForQueue, waitingEntryByData, waitEntry)
					} else {
						q.Add(waitEntry.data.(T))
					}
				default:
					drained = true
				}
			}
		}
	}
}

```

## 限速队列

限速队列应用得非常广泛，比如在我们做一些操作失败后希望重试几次，但是立刻重试很有可能还是会失败，这个时候我们可以延迟一段时间再重试，而且失败次数越多延迟时间越长，这个其实就是限速。首先我们需要来了解下**限速器**。

```go
type TypedRateLimiter[T comparable] interface {
	// When gets an item and gets to decide how long that item should wait
	// 获取item元素应该等待多长时间
	When(item T) time.Duration
	// Forget indicates that an item is finished being retried.  Doesn't matter whether it's for failing
	// or for success, we'll stop tracking it
	 // 表示元素已经完成了重试，不管是成功还是失败都会停止跟踪，也就是抛弃该元素
	Forget(item T)
	// NumRequeues returns back how many failures the item has had
	// 返回元素失败的次数（也就是放入队列的次数）
	NumRequeues(item T) int
}
```

### 1.TypedBucketRateLimiter限速器是利用 `golang.org/x/time/rate` 包中的 `Limiter` 来实现稳定速率(`qps`)的限速器，对应的结构体如下所示：

```go
// TypedBucketRateLimiter adapts a standard bucket to the workqueue ratelimiter API
type TypedBucketRateLimiter[T comparable] struct {
	*rate.Limiter
}
func (r *TypedBucketRateLimiter[T]) When(item T) time.Duration {
	return r.Limiter.Reserve().Delay()
}

func (r *TypedBucketRateLimiter[T]) NumRequeues(item T) int {
	return 0
}

func (r *TypedBucketRateLimiter[T]) Forget(item T) {
}

```

<figure><img src="../../.gitbook/assets/image (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

令牌桶算法内部实现了一个存放token（令牌）的“桶”，初始时“桶”是空的，token会以固定速率往“桶”里填充，直到将其填满为止，多余的token会被丢弃。每个元素都会从令牌桶得到一个token，只有得到token的元素才允许通过（accept），而没有得到token的元素处于等待状态。令牌桶算法通过控制发放token来达到限速目的。

### 2.`TypedItemExponentialFailureRateLimiter` 是比较常用的限速器，他会根据元素错误次数逐渐累加等待时间，定义如下所示：

```go
// TypedItemExponentialFailureRateLimiter does a simple baseDelay*2^<num-failures> limit
// dealing with max failures and expiration are up to the caller
type TypedItemExponentialFailureRateLimiter[T comparable] struct {
	failuresLock sync.Mutex
	failures     map[T]int

	baseDelay time.Duration
	maxDelay  time.Duration
}

func (r *TypedItemExponentialFailureRateLimiter[T]) When(item T) time.Duration {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	exp := r.failures[item]
	r.failures[item] = r.failures[item] + 1

	// The backoff is capped such that 'calculated' value never overflows.
	backoff := float64(r.baseDelay.Nanoseconds()) * math.Pow(2, float64(exp))
	if backoff > math.MaxInt64 {
		return r.maxDelay
	}

	calculated := time.Duration(backoff)
	if calculated > r.maxDelay {
		return r.maxDelay
	}

	return calculated
}

func (r *TypedItemExponentialFailureRateLimiter[T]) NumRequeues(item T) int {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	return r.failures[item]
}

func (r *TypedItemExponentialFailureRateLimiter[T]) Forget(item T) {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	delete(r.failures, item)
}
```

排队指数算法将相同元素的排队数作为指数，排队数增大，速率限制呈指数级增长，但其最大值不会超过maxDelay。

<figure><img src="../../.gitbook/assets/image (2) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

### 3.`TypedItemFastSlowRateLimiter` 和上面的指数级限速器很像，都是用于错误尝试的，但是二者的限速策略不同，`ItemFastSlowRateLimiter` 是尝试次数超过阈值后用长延迟，否则用短延迟，具体的实现如下所示：

```go
// TypedItemFastSlowRateLimiter does a quick retry for a certain number of attempts, then a slow retry after that
type TypedItemFastSlowRateLimiter[T comparable] struct {
    failuresLock sync.Mutex
    failures     map[T]int

    maxFastAttempts int
    fastDelay       time.Duration
    slowDelay       time.Duration
}

func (r *TypedItemFastSlowRateLimiter[T]) When(item T) time.Duration {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	r.failures[item] = r.failures[item] + 1

	if r.failures[item] <= r.maxFastAttempts {
		return r.fastDelay
	}

	return r.slowDelay
}

func (r *TypedItemFastSlowRateLimiter[T]) NumRequeues(item T) int {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	return r.failures[item]
}

func (r *TypedItemFastSlowRateLimiter[T]) Forget(item T) {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	delete(r.failures, item)
}
```

计数器算法是限速算法中最简单的一种，其原理是：限制一段时间内允许通过的元素数量，例如在1分钟内只允许通过100个元素，每插入一个元素，计数器自增1，当计数器数到100的阈值且还在限速周期内时，则不允许元素再通过。但WorkQueue在此基础上扩展了fast和slow速率。

计数器算法提供了4个主要字段：failures、fastDelay、slowDelay及maxFastAttempts。其中，failures字段用于统计元素排队数，每当AddRateLimited方法插入新元素时，会为该字段加1；而fastDelay和slowDelay字段是用于定义fast、slow速率的；另外，maxFastAttempts字段用于控制从fast速率转换到slow速率。计数器算法核心实现的代码示例如下：

### 4.`TypedMaxOfRateLimiter` 限速器内部是一个限速器 slice，每次返回所有限速器里面延迟最大的一个限速器，具体的实现如下所示：

```go
// TypedMaxOfRateLimiter calls every RateLimiter and returns the worst case response
// When used with a token bucket limiter, the burst could be apparently exceeded in cases where particular items
// were separately delayed a longer time.
type TypedMaxOfRateLimiter[T comparable] struct {
	limiters []TypedRateLimiter[T]
}

// client-go/util/workqueue/default_rate_limiters.go
func (r *MaxOfRateLimiter) When(item interface{}) time.Duration {
    ret := time.Duration(0)
    // 获取所有限速器里面时间最大的
    for _, limiter := range r.limiters {
        curr := limiter.When(item)
        if curr > ret {
            ret = curr
        }
    }
    return ret
}

func (r *MaxOfRateLimiter) NumRequeues(item interface{}) int {
    ret := 0
    // 同样获取所有限速器里面最大的 Requeue 次数
    for _, limiter := range r.limiters {
        curr := limiter.NumRequeues(item)
        if curr > ret {
            ret = curr
        }
    }
    return ret
}

func (r *MaxOfRateLimiter) Forget(item interface{}) {
    // 调用所有的限速器的 Forget 方法
    for _, limiter := range r.limiters {
        limiter.Forget(item)
    }
}

```

### 5.`TypedWithMaxWaitRateLimiter`可以设置最长超时

<pre class="language-go"><code class="lang-go">// TypedWithMaxWaitRateLimiter have maxDelay which avoids waiting too long
type TypedWithMaxWaitRateLimiter[T comparable] struct {
	limiter  TypedRateLimiter[T]
	maxDelay time.Duration
}

<strong>func (w TypedWithMaxWaitRateLimiter[T]) When(item T) time.Duration {
</strong>	delay := w.limiter.When(item)
	if delay > w.maxDelay {
		return w.maxDelay
	}

	return delay
}

func (w TypedWithMaxWaitRateLimiter[T]) Forget(item T) {
	w.limiter.Forget(item)
}

func (w TypedWithMaxWaitRateLimiter[T]) NumRequeues(item T) int {
	return w.limiter.NumRequeues(item)
}

</code></pre>
