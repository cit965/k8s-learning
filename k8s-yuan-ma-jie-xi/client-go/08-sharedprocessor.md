# 08-sharedProcessor

### sharedIndexInformer

`sharedIndexInformer` 相比普通的 informer 来说, 可以共享 reflector 反射器, 业务代码可以注册多个 resourceEventHandler 方法, 无需重复创建 informer 做监听及事件注册.



如果相同资源实例化多个 informer, 那么每个 informer 都有一个 reflector 和 store. 不仅会有数据序列化的开销, 而且缓存 store 不能复用, 可能一个对象存在多个 informer 的 store 里.



下面 `sharedIndexInformer` 简化的实现原理架构图.

<figure><img src="../../.gitbook/assets/image (5) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

那 sharedIndexInformer 如何去通知多个 控制器呢？ 这里就用到了 shareProcessor



```go
// `*sharedIndexInformer` implements SharedIndexInformer and has three
// main components.  One is an indexed local cache, `indexer Indexer`.
// The second main component is a Controller that pulls
// objects/notifications using the ListerWatcher and pushes them into
// a DeltaFIFO --- whose knownObjects is the informer's local cache
// --- while concurrently Popping Deltas values from that fifo and
// processing them with `sharedIndexInformer::HandleDeltas`.  Each
// invocation of HandleDeltas, which is done with the fifo's lock
// held, processes each Delta in turn.  For each Delta this both
// updates the local cache and stuffs the relevant notification into
// the sharedProcessor.  The third main component is that
// sharedProcessor, which is responsible for relaying those
// notifications to each of the informer's clients.
type sharedIndexInformer struct {
	indexer    Indexer
	controller Controller

	// 关键的共享 process 来了
	processor             *sharedProcessor
	cacheMutationDetector MutationDetector

	listerWatcher ListerWatcher

	// objectType is an example object of the type this informer is expected to handle. If set, an event
	// with an object with a mismatching type is dropped instead of being delivered to listeners.
	objectType runtime.Object

	...
}
```

#### 添加 eventHandler



`sharedIndexInformer` 是支持动态添加 `ResourceEventHandler` 事件方法.

根据传入的 handler 对象构建 listener 监听器. 然后把监听器加到 listeners 数组里, 并启动 run 和 pop 两个协程.

```go
func (s *sharedIndexInformer) AddEventHandler(handler ResourceEventHandler) (ResourceEventHandlerRegistration, error) {
    return s.AddEventHandlerWithResyncPeriod(handler, s.defaultEventHandlerResyncPeriod)
}

const minimumResyncPeriod = 1 * time.Second

func (s *sharedIndexInformer) AddEventHandlerWithResyncPeriod(handler ResourceEventHandler, resyncPeriod time.Duration) (ResourceEventHandlerRegistration, error) {
	s.startedLock.Lock()
	defer s.startedLock.Unlock()

	if s.stopped {
		return nil, fmt.Errorf("handler %v was not added to shared informer because it has stopped already", handler)
	}

	if resyncPeriod > 0 {
		if resyncPeriod < minimumResyncPeriod {
			klog.Warningf("resyncPeriod %v is too small. Changing it to the minimum allowed value of %v", resyncPeriod, minimumResyncPeriod)
			resyncPeriod = minimumResyncPeriod
		}

		if resyncPeriod < s.resyncCheckPeriod {
			if s.started {
				klog.Warningf("resyncPeriod %v is smaller than resyncCheckPeriod %v and the informer has already started. Changing it to %v", resyncPeriod, s.resyncCheckPeriod, s.resyncCheckPeriod)
				resyncPeriod = s.resyncCheckPeriod
			} else {
				// if the event handler's resyncPeriod is smaller than the current resyncCheckPeriod, update
				// resyncCheckPeriod to match resyncPeriod and adjust the resync periods of all the listeners
				// accordingly
				s.resyncCheckPeriod = resyncPeriod
				s.processor.resyncCheckPeriodChanged(resyncPeriod)
			}
		}
	}
	// 实例化一个 listener 监听对象
	listener := newProcessListener(handler, resyncPeriod, determineResyncPeriod(resyncPeriod, s.resyncCheckPeriod), s.clock.Now(), initialBufferSize, s.HasSynced)
	// 如果未启动，把构建的 listener 放到 processor 的 listeners 数组里,并启动两个协程处理 run 和 pop 方法.
	if !s.started {
		return s.processor.addListener(listener), nil
	}
	// 如果已经启动，需要安全的加入进去
	// in order to safely join, we have to
	// 1. stop sending add/update/delete notifications
	// 2. do a list against the store
	// 3. send synthetic "Add" events to the new handler
	// 4. unblock
	s.blockDeltas.Lock()
	defer s.blockDeltas.Unlock()

	handle := s.processor.addListener(listener)
	for _, item := range s.indexer.List() {
		// Note that we enqueue these notifications with the lock held
		// and before returning the handle. That means there is never a
		// chance for anyone to call the handle's HasSynced method in a
		// state when it would falsely return true (i.e., when the
		// shared informer is synced but it has not observed an Add
		// with isInitialList being true, nor when the thread
		// processing notifications somehow goes faster than this
		// thread adding them and the counter is temporarily zero).
		listener.add(addNotification{newObj: item, isInInitialList: true})
	}
	return handle, nil
}
```

#### HandleDeltas 核心处理函数

`HandleDeltas` 用来处理从 `DeltaFIFO` 拿到的 deltas 事件列表, 然后通知给所有的 lisenter 去处理.

`distribute` 收到变更的事件后, 遍历通知给所有的 listener 监听器, 这里的通知是把事件写到 listener 的 addCh 通道.

```go
func (s *sharedIndexInformer) HandleDeltas(obj interface{}, isInInitialList bool) error {
	s.blockDeltas.Lock()
	defer s.blockDeltas.Unlock()

	if deltas, ok := obj.(Deltas); ok {
		return processDeltas(s, s.indexer, deltas, isInInitialList)
	}
	return errors.New("object given as Process argument is not Deltas")
}


// 在 sharedIndexInformer 设计里, 这里的 store 是 indexer 索引存储, handler 则是 sharedIndexInformer 自身实现的 ResourceEventHandler 接口.
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

// Conforms to ResourceEventHandler
func (s *sharedIndexInformer) OnAdd(obj interface{}, isInInitialList bool) {
	// Invocation of this function is locked under s.blockDeltas, so it is
	// save to distribute the notification
	s.cacheMutationDetector.AddObject(obj)
	s.processor.distribute(addNotification{newObj: obj, isInInitialList: isInInitialList}, false)
}

// Conforms to ResourceEventHandler
func (s *sharedIndexInformer) OnDelete(old interface{}) {
	// Invocation of this function is locked under s.blockDeltas, so it is
	// save to distribute the notification
	s.processor.distribute(deleteNotification{oldObj: old}, false)
}

func (p *sharedProcessor) distribute(obj interface{}, sync bool) {
	p.listenersLock.RLock()
	defer p.listenersLock.RUnlock()

	for listener, isSyncing := range p.listeners {
		switch {
		case !sync:
			// non-sync messages are delivered to every listener
			listener.add(obj)
		case isSyncing:
			// sync messages are delivered to every syncing listener
			listener.add(obj)
		default:
			// skipping a sync obj for a non-syncing listener
		}
	}
}

```

## processorListener&#x20;

事件添加通过 addCh 通道接受，notification 就是事件，也就是从 DeltaFIFO 弹出的 Deltas，addCh 是一个无缓冲通道，所以可以将其看作一个事件分发器，从 DeltaFIFO 弹出的对象需要逐一送到多个处理器，如果处理器没有及时处理 addCh 则会被阻塞。

```go
func (p *processorListener) add(notification interface{}) {
    if a, ok := notification.(addNotification); ok && a.isInInitialList {
       p.syncTracker.Start()
    }
    p.addCh <- notification
}
```

前面有说在添加 listener 监听器时, 启动两个协程去执行 pop() 和 run().

run() 和 pop() 是 processorListener 的两个最核心的函数，processorListener 就是实现了事件的缓冲和处理，在没有事件的时候可以阻塞处理器，当事件较多是可以把事件缓冲起来，实现了事件分发器与处理器的异步处理。

* `pop()` 监听 addCh 队列把 notification 对象扔到 nextCh 管道里.
* `run()` 对 nextCh 进行监听, 然后根据不同类型调用不同的 `ResourceEventHandler` 方法.

```go
// pop：利用 golang select 来同时处理多个 channel，直到至少有一个 case 表达式满足条件为止。
func (p *processorListener) pop() {
	defer utilruntime.HandleCrash()
	defer close(p.nextCh) // Tell .run() to stop

	var nextCh chan<- interface{}
	var notification interface{}
	for {
		select {
		// 把从 addCh 获取的对象扔到 nextCh 里
		case nextCh <- notification:
			// Notification dispatched
			var ok bool
			notification, ok = p.pendingNotifications.ReadOne()
			if !ok { // Nothing to pop
				nextCh = nil // Disable this select case
			}
		 // 从 addCh 获取对象, 如果上一次的 noti 还未扔到 nextCh 里, 那么之后的对象扔到 buffer 里	
		case notificationToAdd, ok := <-p.addCh:
			if !ok {
				return
			}
			if notification == nil { 
				// pendingNotifications 为空，则说明没有notification 去pop
				// No notification to pop (and pendingNotifications is empty)
				// Optimize the case - skip adding to pendingNotifications
				notification = notificationToAdd
				nextCh = p.nextCh
			} else { 
				// There is already a notification waiting to be dispatched
				// 上一个事件还没发送完成（已经有一个通知等待发送），就先放到缓冲通道中
				p.pendingNotifications.WriteOne(notificationToAdd)
			}
		}
	}
}
```

```go
func (p *processorListener) run() {
    // this call blocks until the channel is closed.  When a panic happens during the notification
    // we will catch it, **the offending item will be skipped!**, and after a short delay (one second)
    // the next notification will be attempted.  This is usually better than the alternative of never
    // delivering again.
    stopCh := make(chan struct{})
    wait.Until(func() {
       for next := range p.nextCh {
          switch notification := next.(type) {
          case updateNotification:
             p.handler.OnUpdate(notification.oldObj, notification.newObj)
          case addNotification:
             p.handler.OnAdd(notification.newObj, notification.isInInitialList)
             if notification.isInInitialList {
                p.syncTracker.Finished()
             }
          case deleteNotification:
             p.handler.OnDelete(notification.oldObj)
          default:
             utilruntime.HandleError(fmt.Errorf("unrecognized notification: %T", next))
          }
       }
       // the only way to get here is if the p.nextCh is empty and closed
       close(stopCh)
    }, 1*time.Second, stopCh)
}
```

## ps

有些人可能理解不了pop的两种case，我这里和群里的兄弟们讨论了下，得出结果

**case1：**

将一个 notification 直接推送给 nextCh 后尝试从 pendingNotifications 里读取，如果没有数据，把 nextCh 设置为空，下次直接走case2，如果有数据，继续执行case1，直到 pendingNotifications里没有数据

**case2：**

从 addCh里获取数据，如果没有数据，直接啥也不做

如果发现 notification为空（此时 case1 是关闭的），重新启动case1 (也就是将notification赋值，将nextCh 开启)

如果发现 不为空，说明case1正在执行，直接放入 pendingNotifications 里排队

