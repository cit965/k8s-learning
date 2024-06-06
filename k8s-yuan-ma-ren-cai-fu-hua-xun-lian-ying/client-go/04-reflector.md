# 04-Reflector

Informer 通过对 APIServer 的资源对象执行 List 和 Watch 操作，把获取到的数据存储在本地的缓存中，其中实现这个的核心功能就是 Reflector，我们可以称其为**反射器**，从名字我们可以看出来它的主要功能就是反射，就是**将 Etcd 里面的数据(Api Server的数据)反射到本地存储（DeltaFIFO）中。**&#x20;



本篇文章基于 k8s源码v1.30.0讲解

### 结构体定义

```go
// Reflector watches a specified resource and causes all changes to be reflected in the given store.
type Reflector struct {
	// name 标识这个反射器的名称，默认为 文件:行数（比如reflector.go:125）
	name string
	// The name of the type we expect to place in the store. The name
	// will be the stringification of expectedGVK if provided, and the
	// stringification of expectedType otherwise. It is for display
	// only, and should not be used for parsing or comparison.
	typeDescription string
	// An example object of the type we expect to place in the store.
	// Only the type needs to be right, except that when that is
	// `unstructured.Unstructured` the object's `"apiVersion"` and
	// `"kind"` must also be right.
	expectedType reflect.Type
	// The GVK of the object we expect to place in the store if unstructured.
	expectedGVK *schema.GroupVersionKind
	// The destination to sync up with the watch source
	store Store
	// listerWatcher is used to perform lists and watches.
	listerWatcher ListerWatcher
	// backoff manages backoff of ListWatch
	backoffManager wait.BackoffManager
	resyncPeriod   time.Duration
	// minWatchTimeout defines the minimum timeout for watch requests.
	minWatchTimeout time.Duration
	// clock allows tests to manipulate time
	clock clock.Clock
	// paginatedResult defines whether pagination should be forced for list calls.
	// It is set based on the result of the initial list call.
	paginatedResult bool
	// lastSyncResourceVersion is the resource version token last
	// observed when doing a sync with the underlying store
	// it is thread safe, but not synchronized with the underlying store
	lastSyncResourceVersion string
	// isLastSyncResourceVersionUnavailable is true if the previous list or watch request with
	// lastSyncResourceVersion failed with an "expired" or "too large resource version" error.
	isLastSyncResourceVersionUnavailable bool
	// lastSyncResourceVersionMutex guards read/write access to lastSyncResourceVersion
	lastSyncResourceVersionMutex sync.RWMutex
	// Called whenever the ListAndWatch drops the connection with an error.
	watchErrorHandler WatchErrorHandler
	// WatchListPageSize is the requested chunk size of initial and resync watch lists.
	// If unset, for consistent reads (RV="") or reads that opt-into arbitrarily old data
	// (RV="0") it will default to pager.PageSize, for the rest (RV != "" && RV != "0")
	// it will turn off pagination to allow serving them from watch cache.
	// NOTE: It should be used carefully as paginated lists are always served directly from
	// etcd, which is significantly less efficient and may lead to serious performance and
	// scalability problems.
	WatchListPageSize int64
	// ShouldResync is invoked periodically and whenever it returns `true` the Store's Resync operation is invoked
	ShouldResync func() bool
	// MaxInternalErrorRetryDuration defines how long we should retry internal errors returned by watch.
	MaxInternalErrorRetryDuration time.Duration
	// UseWatchList if turned on instructs the reflector to open a stream to bring data from the API server.
	// Streaming has the primary advantage of using fewer server's resources to fetch data.
	//
	// The old behaviour establishes a LIST request which gets data in chunks.
	// Paginated list is less efficient and depending on the actual size of objects
	// might result in an increased memory consumption of the APIServer.
	//
	// See https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/3157-watch-list#design-details
	//
	// TODO(#115478): Consider making reflector.UseWatchList a private field. Since we implemented "api streaming" on the etcd storage layer it should work.
	UseWatchList *bool
}
```

### 初始化

**通过 NewReflector 实例化反射器的时候，必须传入一个 ListerWatcher 接口对象，这个也是反射器最核心的功能**，该接口拥有 List 和 Watch 方法，用于获取和监控资源对象。

```go
// NewReflector creates a new Reflector with its name defaulted to the closest source_file.go:line in the call stack
// that is outside this package. See NewReflectorWithOptions for further information.
func NewReflector(lw ListerWatcher, expectedType interface{}, store Store, resyncPeriod time.Duration) *Reflector {
	return NewReflectorWithOptions(lw, expectedType, store, ReflectorOptions{ResyncPeriod: resyncPeriod})
}

// NewReflectorWithOptions creates a new Reflector object which will keep the
// given store up to date with the server's contents for the given
// resource. Reflector promises to only put things in the store that
// have the type of expectedType, unless expectedType is nil. If
// resyncPeriod is non-zero, then the reflector will periodically
// consult its ShouldResync function to determine whether to invoke
// the Store's Resync operation; `ShouldResync==nil` means always
// "yes".  This enables you to use reflectors to periodically process
// everything as well as incrementally processing the things that
// change.
func NewReflectorWithOptions(lw ListerWatcher, expectedType interface{}, store Store, options ReflectorOptions) *Reflector {
	reflectorClock := options.Clock
	if reflectorClock == nil {
		reflectorClock = clock.RealClock{}
	}
	minWatchTimeout := defaultMinWatchTimeout
	if options.MinWatchTimeout > defaultMinWatchTimeout {
		minWatchTimeout = options.MinWatchTimeout
	}
	r := &Reflector{
		name:            options.Name,
		resyncPeriod:    options.ResyncPeriod,
		minWatchTimeout: minWatchTimeout,
		typeDescription: options.TypeDescription,
		listerWatcher:   lw,
		store:           store,
		// We used to make the call every 1sec (1 QPS), the goal here is to achieve ~98% traffic reduction when
		// API server is not healthy. With these parameters, backoff will stop at [30,60) sec interval which is
		// 0.22 QPS. If we don't backoff for 2min, assume API server is healthy and we reset the backoff.
		backoffManager:    wait.NewExponentialBackoffManager(800*time.Millisecond, 30*time.Second, 2*time.Minute, 2.0, 1.0, reflectorClock),
		clock:             reflectorClock,
		watchErrorHandler: WatchErrorHandler(DefaultWatchErrorHandler),
		expectedType:      reflect.TypeOf(expectedType),
	}

	if r.name == "" {
		r.name = naming.GetNameFromCallsite(internalPackages...)
	}

	if r.typeDescription == "" {
		r.typeDescription = getTypeDescriptionFromObject(expectedType)
	}

	if r.expectedGVK == nil {
		r.expectedGVK = getExpectedGVKFromObject(expectedType)
	}

	// don't overwrite UseWatchList if already set
	// because the higher layers (e.g. storage/cacher) disabled it on purpose
	if r.UseWatchList == nil {
		if s := os.Getenv("ENABLE_CLIENT_GO_WATCH_LIST_ALPHA"); len(s) > 0 {
			r.UseWatchList = ptr.To(true)
		}
	}

	return r
}
```

重点需要关注的参数：

* ```
  ListerWatcher ：

  每个资源都需要实现的2种动作，用来从apiserver同步数据
  ```
* ```
  expectedType： 

  期望放到 Store 中的类型，如果是unstructured.Unstructured，其`“apiVersion”`和“Kind”也必须是正确的。
  ```
* ```
  Store：

  Deltafifo 的一个实例
  ```

### 启动

启动 refactor 就是调用 Run方法，Run方法会调动 ListAndWatch 方法



逻辑大体分为三大块：

A.List操作（只执行一次）：&#x20;

（1）设置ListOptions，将ResourceVersion设置为“0”；&#x20;

（2）调用r.listerWatcher.List方法，执行list操作，即获取全量的资源对象；&#x20;

（3）根据list回来的资源对象，获取最新的resourceVersion；&#x20;

（4）资源转换，将list操作获取回来的结果转换为\[]runtime.Object结构；&#x20;

（5）调用r.syncWith，根据list回来转换后的结果去替换store里的items；&#x20;

（6）调用r.setLastSyncResourceVersion，为Reflector更新已被处理的最新资源对象的resourceVersion值；



B.Resync操作（异步循环执行）；&#x20;

（1）判断是否需要执行Resync操作，即重新同步；&#x20;

（2）需要则调用r.store.Resync操作后端store做处理；

C.Watch操作（循环执行）：&#x20;

（1）stopCh处理，判断是否需要退出循环；&#x20;

（2）设置ListOptions，设置resourceVersion为最新的resourceVersion，即从list回来的最新resourceVersion开始执行watch操作；

（3）调用r.listerWatcher.Watch，开始监听操作；&#x20;

（4）watch监听操作的错误返回处理；&#x20;

（5）调用r.watchHandler，处理watch操作返回来的结果，操作后端store，新增、更新或删除items；&#x20;

```go
func (r *Reflector) Run(stopCh <-chan struct{}) {
	klog.V(3).Infof("Starting reflector %s (%s) from %s", r.typeDescription, r.resyncPeriod, r.name)
	wait.BackoffUntil(func() {
		if err := r.ListAndWatch(stopCh); err != nil {
			r.watchErrorHandler(r, err)
		}
	}, r.backoffManager, true, stopCh)
	klog.V(3).Infof("Stopping reflector %s (%s) from %s", r.typeDescription, r.resyncPeriod, r.name)
}

// ListAndWatch first lists all items and get the resource version at the moment of call,
// and then use the resource version to watch.
// It returns error if ListAndWatch didn't even try to initialize watch.
func (r *Reflector) ListAndWatch(stopCh <-chan struct{}) error {
	klog.V(3).Infof("Listing and watching %v from %s", r.typeDescription, r.name)
	var err error
	var w watch.Interface
	useWatchList := ptr.Deref(r.UseWatchList, false)
	fallbackToList := !useWatchList

	if useWatchList {
		w, err = r.watchList(stopCh)
		if w == nil && err == nil {
			// stopCh was closed
			return nil
		}
		if err != nil {
			klog.Warningf("The watchlist request ended with an error, falling back to the standard LIST/WATCH semantics because making progress is better than deadlocking, err = %v", err)
			fallbackToList = true
			// ensure that we won't accidentally pass some garbage down the watch.
			w = nil
		}
	}

	if fallbackToList {
		err = r.list(stopCh)
		if err != nil {
			return err
		}
	}

	klog.V(2).Infof("Caches populated for %v from %s", r.typeDescription, r.name)

	resyncerrc := make(chan error, 1)
	cancelCh := make(chan struct{})
	defer close(cancelCh)
	go r.startResync(stopCh, cancelCh, resyncerrc)
	return r.watch(w, stopCh, resyncerrc)
}


```





#### A.List操作（只执行一次）：

```go
// list simply lists all items and records a resource version obtained from the server at the moment of the call.
// the resource version can be used for further progress notification (aka. watch).
func (r *Reflector) list(stopCh <-chan struct{}) error {
	var resourceVersion string
	options := metav1.ListOptions{ResourceVersion: r.relistResourceVersion()}

	initTrace := trace.New("Reflector ListAndWatch", trace.Field{Key: "name", Value: r.name})
	defer initTrace.LogIfLong(10 * time.Second)
	var list runtime.Object
	var paginatedResult bool
	var err error
	listCh := make(chan struct{}, 1)
	panicCh := make(chan interface{}, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()
		// Attempt to gather list in chunks, if supported by listerWatcher, if not, the first
		// list request will return the full response.
		pager := pager.New(pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return r.listerWatcher.List(opts)
		}))
		switch {
		case r.WatchListPageSize != 0:
			pager.PageSize = r.WatchListPageSize
		case r.paginatedResult:
			// We got a paginated result initially. Assume this resource and server honor
			// paging requests (i.e. watch cache is probably disabled) and leave the default
			// pager size set.
		case options.ResourceVersion != "" && options.ResourceVersion != "0":
			// User didn't explicitly request pagination.
			//
			// With ResourceVersion != "", we have a possibility to list from watch cache,
			// but we do that (for ResourceVersion != "0") only if Limit is unset.
			// To avoid thundering herd on etcd (e.g. on master upgrades), we explicitly
			// switch off pagination to force listing from watch cache (if enabled).
			// With the existing semantic of RV (result is at least as fresh as provided RV),
			// this is correct and doesn't lead to going back in time.
			//
			// We also don't turn off pagination for ResourceVersion="0", since watch cache
			// is ignoring Limit in that case anyway, and if watch cache is not enabled
			// we don't introduce regression.
			pager.PageSize = 0
		}

		list, paginatedResult, err = pager.ListWithAlloc(context.Background(), options)
		if isExpiredError(err) || isTooLargeResourceVersionError(err) {
			r.setIsLastSyncResourceVersionUnavailable(true)
			// Retry immediately if the resource version used to list is unavailable.
			// The pager already falls back to full list if paginated list calls fail due to an "Expired" error on
			// continuation pages, but the pager might not be enabled, the full list might fail because the
			// resource version it is listing at is expired or the cache may not yet be synced to the provided
			// resource version. So we need to fallback to resourceVersion="" in all to recover and ensure
			// the reflector makes forward progress.
			list, paginatedResult, err = pager.ListWithAlloc(context.Background(), metav1.ListOptions{ResourceVersion: r.relistResourceVersion()})
		}
		close(listCh)
	}()
	select {
	case <-stopCh:
		return nil
	case r := <-panicCh:
		panic(r)
	case <-listCh:
	}
	initTrace.Step("Objects listed", trace.Field{Key: "error", Value: err})
	if err != nil {
		klog.Warningf("%s: failed to list %v: %v", r.name, r.typeDescription, err)
		return fmt.Errorf("failed to list %v: %w", r.typeDescription, err)
	}

	// We check if the list was paginated and if so set the paginatedResult based on that.
	// However, we want to do that only for the initial list (which is the only case
	// when we set ResourceVersion="0"). The reasoning behind it is that later, in some
	// situations we may force listing directly from etcd (by setting ResourceVersion="")
	// which will return paginated result, even if watch cache is enabled. However, in
	// that case, we still want to prefer sending requests to watch cache if possible.
	//
	// Paginated result returned for request with ResourceVersion="0" mean that watch
	// cache is disabled and there are a lot of objects of a given type. In such case,
	// there is no need to prefer listing from watch cache.
	if options.ResourceVersion == "0" && paginatedResult {
		r.paginatedResult = true
	}

	r.setIsLastSyncResourceVersionUnavailable(false) // list was successful
	listMetaInterface, err := meta.ListAccessor(list)
	if err != nil {
		return fmt.Errorf("unable to understand list result %#v: %v", list, err)
	}
	resourceVersion = listMetaInterface.GetResourceVersion()
	initTrace.Step("Resource version extracted")
	items, err := meta.ExtractListWithAlloc(list)
	if err != nil {
		return fmt.Errorf("unable to understand list result %#v (%v)", list, err)
	}
	initTrace.Step("Objects extracted")
	if err := r.syncWith(items, resourceVersion); err != nil {
		return fmt.Errorf("unable to sync list result: %v", err)
	}
	initTrace.Step("SyncWith done")
	r.setLastSyncResourceVersion(resourceVersion)
	initTrace.Step("Resource version updated")
	return nil

```

**B.Resync操作（异步循环执行）；**&#x20;

```go
// startResync periodically calls r.store.Resync() method.
// Note that this method is blocking and should be
// called in a separate goroutine.
func (r *Reflector) startResync(stopCh <-chan struct{}, cancelCh <-chan struct{}, resyncerrc chan error) {
	resyncCh, cleanup := r.resyncChan()
	defer func() {
		cleanup() // Call the last one written into cleanup
	}()
	for {
		select {
		case <-resyncCh:
		case <-stopCh:
			return
		case <-cancelCh:
			return
		}
		if r.ShouldResync == nil || r.ShouldResync() {
			klog.V(4).Infof("%s: forcing resync", r.name)
			if err := r.store.Resync(); err != nil {
				resyncerrc <- err
				return
			}
		}
		cleanup()
		resyncCh, cleanup = r.resyncChan()
	}
}
```

**C.Watch操作（循环执行）：**

```go
// watch simply starts a watch request with the server.
func (r *Reflector) watch(w watch.Interface, stopCh <-chan struct{}, resyncerrc chan error) error {
	var err error
	retry := NewRetryWithDeadline(r.MaxInternalErrorRetryDuration, time.Minute, apierrors.IsInternalError, r.clock)

	for {
		// give the stopCh a chance to stop the loop, even in case of continue statements further down on errors
		select {
		case <-stopCh:
			// we can only end up here when the stopCh
			// was closed after a successful watchlist or list request
			if w != nil {
				w.Stop()
			}
			return nil
		default:
		}

		// start the clock before sending the request, since some proxies won't flush headers until after the first watch event is sent
		start := r.clock.Now()

		if w == nil {
			timeoutSeconds := int64(r.minWatchTimeout.Seconds() * (rand.Float64() + 1.0))
			options := metav1.ListOptions{
				ResourceVersion: r.LastSyncResourceVersion(),
				// We want to avoid situations of hanging watchers. Stop any watchers that do not
				// receive any events within the timeout window.
				TimeoutSeconds: &timeoutSeconds,
				// To reduce load on kube-apiserver on watch restarts, you may enable watch bookmarks.
				// Reflector doesn't assume bookmarks are returned at all (if the server do not support
				// watch bookmarks, it will ignore this field).
				AllowWatchBookmarks: true,
			}

			w, err = r.listerWatcher.Watch(options)
			if err != nil {
				if canRetry := isWatchErrorRetriable(err); canRetry {
					klog.V(4).Infof("%s: watch of %v returned %v - backing off", r.name, r.typeDescription, err)
					select {
					case <-stopCh:
						return nil
					case <-r.backoffManager.Backoff().C():
						continue
					}
				}
				return err
			}
		}

		err = watchHandler(start, w, r.store, r.expectedType, r.expectedGVK, r.name, r.typeDescription, r.setLastSyncResourceVersion, nil, r.clock, resyncerrc, stopCh)
		// Ensure that watch will not be reused across iterations.
		w.Stop()
		w = nil
		retry.After(err)
		if err != nil {
			if err != errorStopRequested {
				switch {
				case isExpiredError(err):
					// Don't set LastSyncResourceVersionUnavailable - LIST call with ResourceVersion=RV already
					// has a semantic that it returns data at least as fresh as provided RV.
					// So first try to LIST with setting RV to resource version of last observed object.
					klog.V(4).Infof("%s: watch of %v closed with: %v", r.name, r.typeDescription, err)
				case apierrors.IsTooManyRequests(err):
					klog.V(2).Infof("%s: watch of %v returned 429 - backing off", r.name, r.typeDescription)
					select {
					case <-stopCh:
						return nil
					case <-r.backoffManager.Backoff().C():
						continue
					}
				case apierrors.IsInternalError(err) && retry.ShouldRetry():
					klog.V(2).Infof("%s: retrying watch of %v internal error: %v", r.name, r.typeDescription, err)
					continue
				default:
					klog.Warningf("%s: watch of %v ended with: %v", r.name, r.typeDescription, err)
				}
			}
			return nil
		}
	}

```
