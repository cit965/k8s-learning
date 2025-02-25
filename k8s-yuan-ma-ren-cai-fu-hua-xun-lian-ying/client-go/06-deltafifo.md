# 06-DeltaFIFO

<figure><img src="../../.gitbook/assets/image (4) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

DeltaFIFO本质上是一个先进先出的队列，有数据的生产者和消费者，其中生产者是Reflector调用的Add方法，消费者是Controller调用的Pop方法。下面分析DeltaFIFO的核心功能：

```go
// DeltaFIFO is like FIFO, but differs in two ways.  One is that the
// accumulator associated with a given object's key is not that object
// but rather a Deltas, which is a slice of Delta values for that
// object.  Applying an object to a Deltas means to append a Delta
// except when the potentially appended Delta is a Deleted and the
// Deltas already ends with a Deleted.  In that case the Deltas does
// not grow, although the terminal Deleted will be replaced by the new
// Deleted if the older Deleted's object is a
// DeletedFinalStateUnknown.
//
// The other difference is that DeltaFIFO has two additional ways that
// an object can be applied to an accumulator: Replaced and Sync.
// If EmitDeltaTypeReplaced is not set to true, Sync will be used in
// replace events for backwards compatibility.  Sync is used for periodic
// resync events.
//
// DeltaFIFO is a producer-consumer queue, where a Reflector is
// intended to be the producer, and the consumer is whatever calls
// the Pop() method.
//
// DeltaFIFO solves this use case:
//   - You want to process every object change (delta) at most once.
//   - When you process an object, you want to see everything
//     that's happened to it since you last processed it.
//   - You want to process the deletion of some of the objects.
//   - You might want to periodically reprocess objects.
//
// DeltaFIFO's Pop(), Get(), and GetByKey() methods return
// interface{} to satisfy the Store/Queue interfaces, but they
// will always return an object of type Deltas. List() returns
// the newest object from each accumulator in the FIFO.
//
// A DeltaFIFO's knownObjects KeyListerGetter provides the abilities
// to list Store keys and to get objects by Store key.  The objects in
// question are called "known objects" and this set of objects
// modifies the behavior of the Delete, Replace, and Resync methods
// (each in a different way).
//
// A note on threading: If you call Pop() in parallel from multiple
// threads, you could end up with multiple threads processing slightly
// different versions of the same object.
type DeltaFIFO struct {
	// lock/cond protects access to 'items' and 'queue'.
	lock sync.RWMutex
	cond sync.Cond

	// `items` maps a key to a Deltas.
	// Each such Deltas has at least one Delta.
	items map[string]Deltas

	// `queue` maintains FIFO order of keys for consumption in Pop().
	// There are no duplicates in `queue`.
	// A key is in `queue` if and only if it is in `items`.
	queue []string

   // 字段省略
}

// Deltas is a list of one or more 'Delta's to an individual object.
// The oldest delta is at index 0, the newest delta is the last one.
type Deltas []Delta

// Delta is a member of Deltas (a list of Delta objects) which
// in its turn is the type stored by a DeltaFIFO. It tells you what
// change happened, and the object's state after* that change.
//
// [*] Unless the change is a deletion, and then you'll get the final
// state of the object before it was deleted.
type Delta struct {
	Type   DeltaType
	Object interface{}
}


// DeltaType is the type of a change (addition, deletion, etc)
type DeltaType string

// Change type definition
const (
	Added   DeltaType = "Added"
	Updated DeltaType = "Updated"
	Deleted DeltaType = "Deleted"
	// Replaced is emitted when we encountered watch errors and had to do a
	// relist. We don't know if the replaced object has changed.
	//
	// NOTE: Previous versions of DeltaFIFO would use Sync for Replace events
	// as well. Hence, Replaced is only emitted when the option
	// EmitDeltaTypeReplaced is true.
	Replaced DeltaType = "Replaced"
	// Sync is for synthetic events during a periodic resync.
	Sync DeltaType = "Sync"
)
```

从 DeltaFIFO 数据结构来看，里头存储着 map\[obj key]Deltas 和 object queue。Delta 装有对象数据及对象的变化类型。其中 DeltaType 就是资源变化的类型， 比如 Add、Update 等；Delta Object 就是具体的 Kubernetes 资源对象， 如pod等资源对象。

DeltaFIFO与其他队列最大的不同之处是，它会保留所有关于资源对象（obj）的操作类型，队列中会存在拥有不同操作类型的同一个资源对象，消费者在处理该资源对象时能够了解该资源对象所发生的事情。

queue字段存储资源对象的key，该key通过KeyOf函数计算得到。items字段通过map数据结构的方式存储，value存储的是对象的Deltas数组。DeltaFIFO存储结构如图所示。

<figure><img src="../../.gitbook/assets/image (3) (1) (1) (1) (1) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

### 生产者

DeltaFIFO队列中的资源对象在Added（资源添加）事件、Updated（资源更新）事件、Deleted（资源删除）事件中都调用了queueActionLocked函数，它是DeltaFIFO实现的关键:

```go
// Add inserts an item, and puts it in the queue. The item is only enqueued
// if it doesn't already exist in the set.
func (f *DeltaFIFO) Add(obj interface{}) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.populated = true
	return f.queueActionLocked(Added, obj)
}

// Update is just like Add, but makes an Updated Delta.
func (f *DeltaFIFO) Update(obj interface{}) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.populated = true
	return f.queueActionLocked(Updated, obj)
}

// queueActionLocked appends to the delta list for the object.
// Caller must lock first.
func (f *DeltaFIFO) queueActionLocked(actionType DeltaType, obj interface{}) error {
	return f.queueActionInternalLocked(actionType, actionType, obj)
}

// queueActionInternalLocked appends to the delta list for the object.
// The actionType is emitted and must honor emitDeltaTypeReplaced.
// The internalActionType is only used within this function and must
// ignore emitDeltaTypeReplaced.
// Caller must lock first.
func (f *DeltaFIFO) queueActionInternalLocked(actionType, internalActionType DeltaType, obj interface{}) error {
	//（1）计算出对象的key
	id, err := f.KeyOf(obj)
	if err != nil {
		return KeyError{obj, err}
	}

	// Every object comes through this code path once, so this is a good
	// place to call the transform func.
	//
	// If obj is a DeletedFinalStateUnknown tombstone or the action is a Sync,
	// then the object have already gone through the transformer.
	//
	// If the objects already present in the cache are passed to Replace(),
	// the transformer must be idempotent to avoid re-mutating them,
	// or coordinate with all readers from the cache to avoid data races.
	// Default informers do not pass existing objects to Replace.
	if f.transformer != nil {
		_, isTombstone := obj.(DeletedFinalStateUnknown)
		if !isTombstone && internalActionType != Sync {
			var err error
			obj, err = f.transformer(obj)
			if err != nil {
				return err
			}
		}
	}
	//（2）构造新的Delta，将新的Delta追加到Deltas末尾
	oldDeltas := f.items[id]
	newDeltas := append(oldDeltas, Delta{actionType, obj})
	//（3）调用dedupDeltas将Delta去重（目前只将Deltas最末尾的两个delete类型的Delta去重）
	newDeltas = dedupDeltas(newDeltas)

	if len(newDeltas) > 0 {
	//（4）判断对象的key是否在queue中，不在则添加入queue中
		if _, exists := f.items[id]; !exists {
			f.queue = append(f.queue, id)
		}
		//（5）根据对象key更新items中的Deltas
		f.items[id] = newDeltas
		//（6）通知所有的消费者解除阻塞
		f.cond.Broadcast()
	} else {
		// This never happens, because dedupDeltas never returns an empty list
		// when given a non-empty list (as it is here).
		// If somehow it happens anyway, deal with it but complain.
		if oldDeltas == nil {
			klog.Errorf("Impossible dedupDeltas for id=%q: oldDeltas=%#+v, obj=%#+v; ignoring", id, oldDeltas, obj)
			return nil
		}
		klog.Errorf("Impossible dedupDeltas for id=%q: oldDeltas=%#+v, obj=%#+v; breaking invariant by storing empty Deltas", id, oldDeltas, obj)
		f.items[id] = newDeltas
		return fmt.Errorf("Impossible dedupDeltas for id=%q: oldDeltas=%#+v, obj=%#+v; broke DeltaFIFO invariant by storing empty Deltas", id, oldDeltas, obj)
	}
	return nil
}
```

### 消费者

Pop方法作为消费者方法使用，从DeltaFIFO的头部取出最早进入队列中的资源对象数据。Pop方法须传入process函数，用于接收并处理对象的回调方法，代码示例如下：

<pre class="language-go"><code class="lang-go">// Pop blocks until the queue has some items, and then returns one.  If
// multiple items are ready, they are returned in the order in which they were
// added/updated. The item is removed from the queue (and the store) before it
// is returned, so if you don't successfully process it, you need to add it back
// with AddIfNotPresent().
// process function is called under lock, so it is safe to update data structures
// in it that need to be in sync with the queue (e.g. knownKeys). The PopProcessFunc
// may return an instance of ErrRequeue with a nested error to indicate the current
// item should be requeued (equivalent to calling AddIfNotPresent under the lock).
// process should avoid expensive I/O operation so that other queue operations, i.e.
// Add() and Get(), won't be blocked for too long.
//
// Pop returns a 'Deltas', which has a complete list of all the things
// that happened to the object (deltas) while it was sitting in the queue.
func (f *DeltaFIFO) Pop(process PopProcessFunc) (interface{}, error) {
<strong>	//（1）加锁
</strong>	f.lock.Lock()
	//（9）释放锁
	defer f.lock.Unlock()
	//（2）循环判断queue的长度是否为0，为0则阻塞住，调用f.cond.Wait()，等待通知（与queueActionLocked方法中的f.cond.Broadcast()相对应，即queue中有对象key则发起通知）
	for {
		for len(f.queue) == 0 {
			// When the queue is empty, invocation of Pop() is blocked until new item is enqueued.
			// When Close() is called, the f.closed is set and the condition is broadcasted.
			// Which causes this loop to continue and return from the Pop().
			if f.closed {
				return nil, ErrFIFOClosed
			}

			f.cond.Wait()
		}
		isInInitialList := !f.hasSynced_locked()
		//（3）取出queue的队头对象key
		id := f.queue[0]
		//（4）更新queue，把queue中所有的对象key前移，相当于把第一个对象key给pop出去
		f.queue = f.queue[1:]
		depth := len(f.queue)
		//（5）initialPopulationCount变量减1，当减到0时则说明initialPopulationCount代表第一次调用Replace方法加入DeltaFIFO中的对象key已经被pop完成
		if f.initialPopulationCount > 0 {
			f.initialPopulationCount--
		}
		//（6）根据对象key从items中获取对象
		item, ok := f.items[id]
		if !ok {
			// This should never happen
			klog.Errorf("Inconceivable! %q was in f.queue but not f.items; ignoring.", id)
			continue
		}
		//（7）把对象从items中删除
		delete(f.items, id)
		// Only log traces if the queue depth is greater than 10 and it takes more than
		// 100 milliseconds to process one item from the queue.
		// Queue depth never goes high because processing an item is locking the queue,
		// and new items can't be added until processing finish.
		// https://github.com/kubernetes/kubernetes/issues/103789
		if depth > 10 {
			trace := utiltrace.New("DeltaFIFO Pop Process",
				utiltrace.Field{Key: "ID", Value: id},
				utiltrace.Field{Key: "Depth", Value: depth},
				utiltrace.Field{Key: "Reason", Value: "slow event handlers blocking the queue"})
			defer trace.LogIfLong(100 * time.Millisecond)
		}
		//（8）调用PopProcessFunc处理pop出来的对象
		err := process(item, isInInitialList)
		if e, ok := err.(ErrRequeue); ok {
			f.addIfNotPresent(id, item)
			err = e.Err
		}
		// Don't need to copyDeltas here, because we're transferring
		// ownership to the caller.
		return item, err
	}
}
</code></pre>

当队列中没有数据时，通过f.cond.wait阻塞等待数据，只有收到cond.Broadcast时才说明有数据被添加，解除当前阻塞状态。如果队列中不为空，取出f.queue的头部数据，将该对象传入process回调函数，由上层消费者进行处理。如果process回调函数处理出错，则将该对象重新存入队列。

Controller的processLoop方法负责从DeltaFIFO队列中取出数据传递给process回调函数。process回调函数代码示例如下：

```go
func (s *sharedIndexInformer) HandleDeltas(obj interface{}, isInInitialList bool) error {
	s.blockDeltas.Lock()
	defer s.blockDeltas.Unlock()

	if deltas, ok := obj.(Deltas); ok {
		return processDeltas(s, s.indexer, deltas, isInInitialList)
	}
	return errors.New("object given as Process argument is not Deltas")
}

// Multiplexes updates in the form of a list of Deltas into a Store, and informs
// a given handler of events OnUpdate, OnAdd, OnDelete
func processDeltas(
	// Object which receives event notifications from the given deltas
	handler ResourceEventHandler,
	clientState Store,
	deltas Deltas,
	isInInitialList bool,
) error {
	// from oldest to newest
	for _, d := range deltas {
		obj := d.Object

		switch d.Type {
		case Sync, Replaced, Added, Updated:
			if old, exists, err := clientState.Get(obj); err == nil && exists {
				if err := clientState.Update(obj); err != nil {
					return err
				}
				handler.OnUpdate(old, obj)
			} else {
				if err := clientState.Add(obj); err != nil {
					return err
				}
				handler.OnAdd(obj, isInInitialList)
			}
		case Deleted:
			if err := clientState.Delete(obj); err != nil {
				return err
			}
			handler.OnDelete(obj)
		}
	}
	return nil
}

```

HandleDeltas函数作为process回调函数，当资源对象的操作类型为Added、Updated、Deleted时，将该资源对象存储至Indexer（它是并发安全的存储），并通过distribute函数将资源对象分发至SharedInformer。还记得Informers Example代码示例吗？在Informers Example代码示例中，我们通过informer.AddEventHandler函数添加了对资源事件进行处理的函数，distribute函数则将资源对象分发到该事件处理函数中。

### resync

Resync机制会将Indexer本地存储中的资源对象同步到DeltaFIFO中，并将这些资源对象设置为Sync的操作类型。Resync函数在Reflector中定时执行，它的执行周期由NewReflector函数传入的resyncPeriod参数设定。
