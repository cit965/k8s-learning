# 01-controller-runtime

## 介绍

Operator 是 Kubernetes 用来拓展其 API 的一种开发范式（Pattern），其核心是定义若干的自定义资源及其对应的资源控制器，当这些资源发生变化时其对应的控制器对变化进行调解（Reconcile），最终使得实际状态与预期状态达成一致。K8s-sigs 推出的 [kubebuilder](https://book.kubebuilder.io/) 是一个用于构建 Operator 应用的框架，和 [Operator-SDK](https://github.com/operator-framework/operator-sdk) 一样都依赖了 [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)，提供了高级 API 和抽象，让开发者更直观地编写操作逻辑，并提供用于快速启动新项目的脚手架和代码生成工具。



controller-runtime 这个包的内容不少，为了在一篇文章里能覆盖到，本文仅选取在构建 Operator 过程中起关键作用的包进行介绍，`envtest`、`scheme`、`certwatcher` 等同样重要的包就不在此提及。下面是我读代码时画的意识流思维导图，有的子项目是组成结构，有的子项目是工作角色，比较自由灵活。我选取了 `Cache`,`Source`, `Handler`, `Client`, `Controller` 和 `Manager` 这六个包

<figure><img src="../../.gitbook/assets/image (11).png" alt=""><figcaption></figcaption></figure>

## WorkFlow

对于控制器而言，资源发生变动的信息均来自于 API Server，从资源发生变动到控制器完成调解需要经过多个模块的处理，大体来说可以用下面的图来表示。

<figure><img src="../../.gitbook/assets/image (12).png" alt=""><figcaption></figcaption></figure>

## Cache <a href="#cache" id="cache"></a>

Cache 包通过 Informers 接口封装了 client-go 中的缓存机制 SharedInformer，为每个资源类型都创建对应的 Informer，通过它们的缓存避免所有请求都直接访问 API Server 导致其可能的不堪重负。SharedInformer 机制在 client-go 中定义，它采用增量同步的方式从 API Server 处“订阅”某类资源的事件，并且将事件的增量更新保存在本地存储（Store）当中，其中典型的存储是 DeltaFIFO。SharedInformer 是 k8s client-go 中的核心机制，几乎所有的客户端应用都绕不开它，之后有空再阅读查看其中细节，这里我们不再展开。

控制流路径大致为 cache.New -> newCache -> internal.NewInformers -> sharedInformers，其中：

* cluster 初始化时通过 `cache.New` 创建集群资源缓存，默认的创建缓存方法可以通过传入自定义的缓存初始化函数进行 Mock，大多数情况下不需要传入自定义的函数；
* 在 Cache 的初始化函数当中可以为每个类型的资源定义细粒度的缓存策略，通过 `cache.Options` 中的 `ByObject` 字段进行配置。在 `manager.Options` 中的 `Cache` 就是负责控制缓存的行为的字段。
* `internal.Informers` 提供了 `Get`、`Peek` 和 `Remove` 方法。其中 Get 方法中调用 Peek，若没有获取到指定的 sharedInformer，会根据配置参数中的 `newInformer` 方法创建出来并且添加到 map 当中留作后用；Peek 方法若无法从 map 中获取到也不会自动启动新的 sharedInformer。
* 通过 `internal.Informers` 获取到 sharedInformer，后续的 Source 包能够将事件处理器与其绑定，将从 API Server 处同步到的对象变更事件转化为控制器循环当中的 `reconcile.Request` 对象。

## Source <a href="#source" id="source"></a>

`Source` 顾名思义是来源，但准确来讲这个来源是请求的来源，也就是在 kubebuilder 中所有的控制器需要实现的 Reconcile 方法的 requests.Request 这一参数的生产者。在 Source 有三种类型，Channel，Informer 和 Func。

其中 [Channel](https://github.com/kubernetes-sigs/controller-runtime/blob/7032a3cc91d2afc4c2d54e4a4891cf75da9f75f5/pkg/source/source.go#L67-L86) 类型主要用于外部事件的处理，例如 Github 的 Webhook，需要用户自行编写外部的 Source 来将通用事件写入到内部的 Channel 当中。

Informer 类型的 Source 在控制器当中最常用，它封装了 client-go 的 cache.Informer 接口，将事件处理器与 informer 进行绑定，用于产生源于集群内部的事件，例如 Pod 的创建等。

```go
// Informer is used to provide a source of events originating inside the cluster from Watches (e.g. Pod Create).
type Informer struct {
	// Informer is the controller-runtime Informer
	Informer cache.Informer
}

var _ Source = &Informer{}

// Start is internal and should be called only by the Controller to register an EventHandler with the Informer
// to enqueue reconcile.Requests.
func (is *Informer) Start(ctx context.Context, handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	prct ...predicate.Predicate) error {
	// Informer should have been specified by the user.
	if is.Informer == nil {
		return fmt.Errorf("must specify Informer.Informer")
	}

	_, err := is.Informer.AddEventHandler(internal.NewEventHandler(ctx, queue, handler, prct).HandlerFuncs())
	if err != nil {
		return err
	}
	return nil
}
```

因为 Source 接口只有一个 Start() 方法，所以 Func 类型只是为了方便将单个函数实现成为这个接口而封装出来的类型，在此不再作更多介绍。

在 Source 的内部实现 (`pkg/internal/source`) 中，它将从 Informer 中获取到的对象转换为 Create，Update，Delete 和 Generic 四类事件，四类事件分别由相应的事件处理器进行下一步的处理。其中，事件处理器在 Handler 包中定义，Informer 在 Cache 包中定义。

## Handler <a href="#handler" id="handler"></a>

```go
// TypedEventHandler is experimental and subject to future change.
type TypedEventHandler[T any] interface {
    // Create is called in response to a create event - e.g. Pod Creation.
    Create(context.Context, event.TypedCreateEvent[T], workqueue.RateLimitingInterface)

    // Update is called in response to an update event -  e.g. Pod Updated.
    Update(context.Context, event.TypedUpdateEvent[T], workqueue.RateLimitingInterface)

    // Delete is called in response to a delete event - e.g. Pod Deleted.
    Delete(context.Context, event.TypedDeleteEvent[T], workqueue.RateLimitingInterface)

    // Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
    // external trigger request - e.g. reconcile Autoscaling, or a Webhook.
    Generic(context.Context, event.TypedGenericEvent[T], workqueue.RateLimitingInterface)
}
```

Source 将事件处理器 EventHandlers 和 Informers 进行绑定，Handlers 将某个某类型资源 A 的事件 Event 转化为某类型资源 B 的事件请求 Request 推入工作队列（`workqueue.RateLimitingInterface`，定义在 client-go 当中），其中 A 通常等于 B，但也存在 A 不等于 B 的情况。下面将两种情况区分介绍。

**1. A == B**

A == B 也就是说产生事件的资源和需要调解的资源类型是相同的，例如用户提交了一个 Pod，那 Pod 的控制器就会接收到这个 Pod 被创建的事件，并对该事件进行调解。这是最普遍的情况，在 `pkg/handler/enqueue.go` 中有该情况的实现。

**2. A != B**

A != B 说明在类型 A 产生的事件要发送给类型 B 的控制器进行调解，这在单一资源/控制器的语境下没有太大的意义，但如果将资源的从属关系也纳入其中就很好解释了：父级资源在子资源发生变更时收到相应的事件，级联地调解自身的状态，进而加速多级资源结构整体的调解速度。例如 ReplicaSet 资源应该监听其拥有的 Pod 资源的事件，当 Pod 状态发生变化时，ReplicaSet 控制器也应该调解 ReplicaSet 资源的状态或配置，以求符合预期。

在 `pkg/handler/enqueue_owner.go` 和 `enqueue_mapped.go` 中有 A != B 时的 handler 方法实现。其中 `enqueue_owner.go` 中为我们实现了“子资源变更，父资源调解”的逻辑，在 kubebuilder 中在 builder 方法下使用 `Owns()` 方法可以声明从属关系，从而让我们的控制器能够调解拥有的其他资源的“此类资源”。

而 `enqueue_mapped.go` 则封装了更为通用的事件处理器方法，能够让用户自定义从 client.Object 到 reconcile.Request 的映射，实现更为灵活的事件入队逻辑。

## Controller <a href="#controller" id="controller"></a>

Controller 控制器是我们要补充编码并最终运行的若干实体，它们负责从 K8s 的控制循环中取回对应资源的事件，并且调用自身的调解函数（也就是我们在编写 Operator 时补充的 Reconcile 函数）完成资源状态对齐的任务。如开头的思维导图所示，我列出了 `Reconcile`，`Workqueue`，`Watches()` 和 `Metadata Projection` 这些子项目，下面分别就这些内容进行介绍。

```go
// Controller implements controller.Controller.
type Controller struct {
    // Name is used to uniquely identify a Controller in tracing, logging and monitoring.  Name is required.
    Name string

    // MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run. Defaults to 1.
    MaxConcurrentReconciles int

    // Reconciler is a function that can be called at any time with the Name / Namespace of an object and
    // ensures that the state of the system matches the state specified in the object.
    // Defaults to the DefaultReconcileFunc.
    Do reconcile.Reconciler

    // RateLimiter is used to limit how frequently requests may be queued into the work queue.
    RateLimiter ratelimiter.RateLimiter

    // NewQueue constructs the queue for this controller once the controller is ready to start.
    // This is a func because the standard Kubernetes work queues start themselves immediately, which
    // leads to goroutine leaks if something calls controller.New repeatedly.
    NewQueue func(controllerName string, rateLimiter ratelimiter.RateLimiter) workqueue.RateLimitingInterface

    // Queue is an listeningQueue that listens for events from Informers and adds object keys to
    // the Queue for processing
    Queue workqueue.RateLimitingInterface

    // mu is used to synchronize Controller setup
    mu sync.Mutex

    // Started is true if the Controller has been Started
    Started bool

    // ctx is the context that was passed to Start() and used when starting watches.
    //
    // According to the docs, contexts should not be stored in a struct: https://golang.org/pkg/context,
    // while we usually always strive to follow best practices, we consider this a legacy case and it should
    // undergo a major refactoring and redesign to allow for context to not be stored in a struct.
    ctx context.Context

    // CacheSyncTimeout refers to the time limit set on waiting for cache to sync
    // Defaults to 2 minutes if not set.
    CacheSyncTimeout time.Duration

    // startWatches maintains a list of sources, handlers, and predicates to start when the controller is started.
    startWatches []source.Source

    // LogConstructor is used to construct a logger to then log messages to users during reconciliation,
    // or for example when a watch is started.
    // Note: LogConstructor has to be able to handle nil requests as we are also using it
    // outside the context of a reconciliation.
    LogConstructor func(request *reconcile.Request) logr.Logger

    // RecoverPanic indicates whether the panic caused by reconcile should be recovered.
    RecoverPanic *bool

    // LeaderElected indicates whether the controller is leader elected or always running.
    LeaderElected *bool
}
```

### Reconcile <a href="#reconcile" id="reconcile"></a>

Reconcile 函数也就是[控制器结构体（pkg/internal/controller）](https://github.com/kubernetes-sigs/controller-runtime/blob/7032a3cc91d2afc4c2d54e4a4891cf75da9f75f5/pkg/internal/controller/controller.go#L41)当中 `Do` 这个字段的具体实现，它接收 `reconcile.Request` 返回 `reconcile.Result`，这两个参数类型都极为简单，从中可以表现出 controller-runtime 的设计者们希望把最简单的接口留给开发者。reconcile.Request 其实就是 `NamespacedName`，reconcile.Result 则包含了两个字段 `Requeue` 和 `RequeueAfter`，分别表示是否重新入队和多久后重新入队。这与后续的工作队列模块相互配合，支持我们实现有计划、有规律的调解重试。

### Workqueue

Workqueue 顾名思义是工作队列，与 Controller 控制器和 Source 事件源相互配合，完成对资源变更事件的有序处理过程。workqueue 是 client-go 中的 `workqueue.RateLimitingInterface` 接口，也就是速率受限的工作队列，限定速率的工作由 `rateLimiter` 接口完成，一个对象需要先经过 rateLimiter 同意才能够顺利入队，速率限定器的逻辑可由用户自行定义，但大部分 K8s 客户端的场景当中，使用默认的速率限定逻辑即可。速率受限的工作队列也在 client-go 中完成定义，之后的文章中有机会再探讨。

### Watches

Watches 方法将某一类对象包装成为 Source，并将其通过事件处理器 Handler 与工作队列进行关联。在 Kubebuilder 当中我们直接使用的方法是 `ControllerManagedBy`，它采用构建者模式返回一个 `Builder` 类型的结构，支持我们链式调用配置方法，最终通过 `Complete` 方法完成控制器的构建。在 Builder 结构体下暴露了若干的方法，其中有 For，Owns 和 Watches 这三个方法用于绑定 Source 和 Handler。For 和 Owns 其实是 Watches 的语法糖，它们分别表示监听某类资源和监听拥有的某类资源（从属关系通过 OwnerReferences 构建），都可以通过 Watches 方法来实现。

Watches 方法接受 `client.Object`，`handler.EventHandler` 和 `WatchesOption` 作为参数，从集群的缓存中拿到某类资源的 Informer 封装为 Source，绑定上事件处理器。handler 包中提供的两个现有的方法分别构成了 For 和 Owns 两个方法对 Watches 封装的语法糖。

Watches 的行为还会收到 Predicates 的影响，Predicates 起过滤作用，用来决定什么事件应该进入工作队列，什么事件不应该进入工作队列。刚开始接触 Controller runtime 时许多开发者经常会遇到资源 Spec 变更后触发调解，控制器更新资源 Status 之后再次触发调解的莫名其妙的死循环，这个情况就是 Predicate 没有正确设置，当资源（包括 Status）发生更新后，资源的 ResourceVersion 会发生变更，但如果不希望 Status 更新后触发调解，可以在 `builder.WithEventFilter()` 中传入预先定义好的 `predicate.GenerationChangedPredicate{}`，这样会过滤掉 ResourceVersion 发生变更的事件。

### Metadata Projection

在 Controller Builder 包中有个类型是 `objectProjection` 表示对象的投影。在调用 For，Owns 和 Watches 三个方法时可以通过末尾的不定长选项参数传入有关投影的配置，builder.OnlyMetadata 就是这样的配置。OnlyMetadata 用来告诉控制器只需要缓存元信息，并且只通过 MetadataClient Watch 元信息格式的资源对象。这对于某类资源对象众多、资源占据空间极大或者只知道资源的 GVK 不知道资源的具体结构等情况是非常有用的。

## Controller Manager

Controller Manager 控制器管理器管理了包括控制器在内的若干可运行接口（Runnable），只要实现了方法 `Start(context.Context) error` 就能够成为 Runnable，上述介绍的若干模块都实现了这个方法，例如 Cache, Source, Controller，还有未提及的 Webhook，HttpServer，LeaderElection 等。管理器自身也实现了 Start 方法，用于在我们的主程序中调用运行。上述所有模块的配置也都可以通过 Manager 的配置进行传入，换句话说，Manager 的配置整合了所有其他模块的配置信息。

Manager 还封装了 Cluster 这个接口，cluster 包含了 `rest.Config`, `runtime.Scheme`, `Cache`, `client.Reader` 和 `meta.RESTMapper` 等包含集群信息的重要字段，Cluster 接口所有的方法都是只读的，也确定了该结构就是单纯用于“信息查阅”的。

## Client

Client 封装了常用的客户端功能，Get 和 List 操作优先从缓存中读取，Create，Update 和 Delete 等写入操作直接与 API Server 进行通信。当然可以在初始化客户端时通过 `client.Options.Cache.DisableFor` 字段配置禁用某些资源类型的缓存，直接从 API Server 读取。
