# 04-scheduler 二次开发

我们讲了一个 Pod 是如何被感知到需要调度和如何被调度的，其中在调度过程中寻找合适的 Node 用的都是内置默认的插件，其实 kube-scheduler 是可以扩展的。那么为什么要扩展呢？原因在于默认的插件算法可能并不能满足你的需要，比如我想要在preFilter 阶段就过滤掉带某些标签的节点，又或者我想根据节点实际的资源使用率来打分而不是当前已经分配的资源，那么默认的 kube-scheduler 是无法满足你的要求的，这时候我们就需要开发一个自己的插件对 kube-scheduler 进行扩展。kube-scheduler 扩展的方式主要有下面两种：

1. 使用现有 kube-scheduler 提供的 extender 机制，进行扩展；这种方式就是开发 filter、score、bind的独立运行程序，这种模式允许你使用任何语言开发，开发的程序可以运行在任何 kube-scheduler 可以通过 http 访问访问到的地方；这种模式的优点就是 kube-scheduler 解耦，无需改变 kube-scheduler 代码，只需要增加配置文件即可；但是缺点也很明显，kube-scheduler 需要通过调用API的方式来访问扩展，所以会产生网络IO，这是很重的操作，会降低调度效率，并且还强依赖外部扩展的稳定性
2. 通过调度框架（Scheduling Framework）进行扩展，Kubernetes v1.15 版本中引入了可插拔架构的调度框架，使得定制调度器这个任务变得更加的容易；这种模式只需要在现有扩展点的某个位置插入自定义的插件即可，还可以关闭默认的插件。这种模式也不需要修改现有的 kube-scheduler 框架代码，跟上一种模式不同的是，他是运行在 kube-scheduler 框架中的，就跟默认的内置插件没有区别，所以不需要 API 调用，稳定性好，效率也高。如下图，每一个箭头都是一个扩展点，每一个扩展点都可以插入自定义插件。本文就根据第二种模式详细讲讲，开发一个自定义的插件需要哪些步骤：

<figure><img src="../.gitbook/assets/image (9).png" alt=""><figcaption></figcaption></figure>

我们先回顾下几个重要的概念：scheduler, framework, registry ，他们之间的关系我用下图来表示：

<figure><img src="../.gitbook/assets/image (10).png" alt=""><figcaption></figcaption></figure>

* **scheduler**

是一个调度器，它实现了整体调度逻辑，如在适当位置执行适当的扩展点，一个 Pod 调度失败了需要做什么处理，记录每一个调度情况暴露出 metric 方便外界对 scheduler 的监控等等。scheduler 是串联整个调度流程的。

* **Profiles**

scheduler 的一个成员，就是如下的一个Map

```go
// pkg/scheduler/profile/profile.go

// Map holds frameworks indexed by scheduler name.
type Map map[string]framework.Framework
```

这个 map 的 key 是 scheduler name, vaule 是 Framework。我们一般在使用 kube-scheduler 时，没有对 kube-scheduler 或他配置文件做修改，此时这个 map 的 key 就只有默认的"default-scheduler"，但是我们现在要开发自己的插件，我们可以在配置文件中定义新的 scheduler name，在这个 scheduler name 中引用自己开发的插件，我们看下下面的配置

```yaml
apiVersion: kubescheduler.config.k8s.io/v1beta2
kind: KubeSchedulerConfiguration
leaderElection:
  leaderElect: true
clientConnection:
  kubeconfig: "/etc/kubernetes/scheduler.conf"

profiles:
- schedulerName: my-scheduler-1
  plugins:
    preFilter:
      enabled:
       // 我们在 preFilter 扩展点的 开发的 zoneLabel 插件
        - name: zoneLabel
        
- schedulerName: my-scheduler-2
  plugins:
    queueSort:
      enabled:
      // 我们在 sort 扩展点开发的 mySort 插件，替代默认的 sort插件
        - name: mySort
```

我们在 profiles 中定义了两个 profile，他们的 schedulerName 分别为 my-scheduler-1 和 my-scheduler-2，当这个配置文件被加载后，Profiles（一个map）的 key 就有两个了，即这两个 schedulerName。 想要根据特定配置文件进行调度的 Pod 可以在其 `.spec.schedulerName` 中包含相应的调度程序名称。



默认情况下，会创建一个名为 `default-scheduler` 的配置文件。此配置文件包括上述默认插件。当声明多个配置文件时，每个配置文件都需要一个唯一的调度程序名称。



如果 Pod 未指定调度程序名称，kube-apiserver 会将其设置为 `default-scheduler` 。因此，应该存在具有此调度程序名称的配置文件来调度这些 pod。

* **Framerowk**

Framework 是一个接口，接口里面定义了一系列方法，这些方法主要是用运行插件的，kube-scheduler 的 frameworkImpl 实现了这个接口

```go
type Framework interface {
   Handle
   
   QueueSortFunc() LessFunc

   RunPreFilterPlugins(ctx context.Context, state *CycleState, pod *v1.Pod) (*PreFilterResult, *Status)

   RunPostFilterPlugins(ctx context.Context, state *CycleState, pod *v1.Pod, filteredNodeStatusMap NodeToStatusMap) (*PostFilterResult, *Status)

   RunPreBindPlugins(ctx context.Context, state *CycleState, pod *v1.Pod, nodeName string) *Status

   RunPostBindPlugins(ctx context.Context, state *CycleState, pod *v1.Pod, nodeName string)

   RunReservePluginsReserve(ctx context.Context, state *CycleState, pod *v1.Pod, nodeName string) *Status

   RunReservePluginsUnreserve(ctx context.Context, state *CycleState, pod *v1.Pod, nodeName string)

   RunPermitPlugins(ctx context.Context, state *CycleState, pod *v1.Pod, nodeName string) *Status

   WaitOnPermit(ctx context.Context, pod *v1.Pod) *Status

   RunBindPlugins(ctx context.Context, state *CycleState, pod *v1.Pod, nodeName string) *Status

   HasFilterPlugins() bool

   HasPostFilterPlugins() bool

   HasScorePlugins() bool

   ListPlugins() *config.Plugins

   ProfileName() string
}
```

frameworkImpl 的成员主要是各个扩展点插件数组，用来存放该扩展点插件。frameworkImpl 实现 Framework 这个接口，可以通过 RunxxxxPlugins() 这样的方法来执行 frameworkImpl 中的插件。

```go
type frameworkImpl struct {
	registry             Registry
	snapshotSharedLister framework.SharedLister
	waitingPods          *waitingPodsMap
	scorePluginWeight    map[string]int
	queueSortPlugins     []framework.QueueSortPlugin
	preFilterPlugins     []framework.PreFilterPlugin
	filterPlugins        []framework.FilterPlugin
	postFilterPlugins    []framework.PostFilterPlugin
	preScorePlugins      []framework.PreScorePlugin
	scorePlugins         []framework.ScorePlugin
	reservePlugins       []framework.ReservePlugin
	preBindPlugins       []framework.PreBindPlugin
	bindPlugins          []framework.BindPlugin
	postBindPlugins      []framework.PostBindPlugin
	permitPlugins        []framework.PermitPlugin
	clientSet       clientset.Interface
	kubeConfig      *restclient.Config
	eventRecorder   events.EventRecorder
	informerFactory informers.SharedInformerFactory
	metricsRecorder *metricsRecorder
	profileName     string
	extenders []framework.Extender
	framework.PodNominator
	parallelizer parallelize.Parallelizer

```

## &#x20;Plugins that apply to multiple extension points <a href="#multiple-profiles" id="multiple-profiles"></a>

配置文件配置中有一个附加字段 `multiPoint` ，它允许跨多个扩展点轻松启用或禁用插件,`multiPoint` 配置的目的是简化用户和管理员在使用自定义配置文件时所需的配置。

考虑一个插件 `MyPlugin` ，它实现 `preScore` 、 `score` 、 `preFilter` 和 `filter` 扩展点。要为其所有可用扩展点启用 `MyPlugin` ，配置文件配置如下所示：

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: multipoint-scheduler
    plugins:
      multiPoint:
        enabled:
        - name: MyPlugin
```

这相当于手动为其所有扩展点启用 `MyPlugin` ，如下所示：

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: non-multipoint-scheduler
    plugins:
      preScore:
        enabled:
        - name: MyPlugin
      score:
        enabled:
        - name: MyPlugin
      preFilter:
        enabled:
        - name: MyPlugin
      filter:
        enabled:
        - name: MyPlugin
```

此处使用 `multiPoint` 的一个好处是，如果 `MyPlugin` 将来实现另一个扩展点， `multiPoint` 配置将自动为新扩展启用它。

可以使用该扩展点的 `disabled` 字段从 `MultiPoint` 扩展中排除特定的扩展点。这适用于禁用默认插件、非默认插件，或使用通配符 ( `'*'` ) 禁用所有插件。禁用 `Score` 和 `PreScore` 的示例如下：

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: non-multipoint-scheduler
    plugins:
      multiPoint:
        enabled:
        - name: 'MyPlugin'
      preScore:
        disabled:
        - name: '*'
      score:
        disabled:
        - name: '*'
```

## registry

registry 中文意思为注册，对于插件来说注册的信息为：插件叫什么，如何创建这个插件。下面是 registry 的结构

```go
type PluginFactory = func(configuration runtime.Object, f framework.Handle) (framework.Plugin, error)

type Registry map[string]PluginFactory
```

registry 是一个 map: key 是插件的名字，value 是 PluginFactory 类型的函数，这个函数返回 framework.Plugin，这个 Plugin 就是我们上面说接口，实现这个接口的对象就可以作为插件被调用。所以 PluginFactory 的作用就是新建一个 Plugin 类型的对象。我们可以像下面这么描述

```go
registry[插件名字]创建插件对象的函数
```

初始化流程：在 scheduler 启动前，遍历这个map，执行这个 map 的 value 代表的函数，将函数返回值写入 frameworkImpl 对应的扩展点数组。

执行某个扩展点插件流程：遍历 frameworkImpl 中这个扩展点数组的所有对象，执行它即可。

内置插件的注册叫 InTreeRegistry, 用户自定义插件的注册叫 OutOfTreeRegistry，在注册所有的插件时只需要将内置插件的 registry 和 用户自定义的 registry 合并在一起。这个流程是通过下面这个函数实现的：

```go
// pkg/scheduler/scheduler.go

func New(client clientset.Interface,
	informerFactory informers.SharedInformerFactory,
	dynInformerFactory dynamicinformer.DynamicSharedInformerFactory,
	recorderFactory profile.RecorderFactory,
	stopCh <-chan struct{},
	opts ...Option) (*Scheduler, error) {

    ...

	options := defaultSchedulerOptions

	for _, opt := range opts {
		opt(&options)
	}

    ...

	registry := frameworkplugins.NewInTreeRegistry()

	if err := registry.Merge(options.frameworkOutOfTreeRegistry); err != nil {
		return nil, err
	}
	
	...
}


// pkg/scheduler/framework/plugins/registry.go

func NewInTreeRegistry() runtime.Registry {
	fts := plfeature.Features{
		EnableReadWriteOncePod:                       feature.DefaultFeatureGate.Enabled(features.ReadWriteOncePod),
		EnableVolumeCapacityPriority:                 feature.DefaultFeatureGate.Enabled(features.VolumeCapacityPriority),
		EnableMinDomainsInPodTopologySpread:          feature.DefaultFeatureGate.Enabled(features.MinDomainsInPodTopologySpread),
		EnableNodeInclusionPolicyInPodTopologySpread: feature.DefaultFeatureGate.Enabled(features.NodeInclusionPolicyInPodTopologySpread),
	}

	return runtime.Registry{
		selectorspread.Name:                  selectorspread.New,
		imagelocality.Name:                   imagelocality.New,
		tainttoleration.Name:                 tainttoleration.New,
		nodename.Name:                        nodename.New,
		nodeports.Name:                       nodeports.New,
		nodeaffinity.Name:                    nodeaffinity.New,
		podtopologyspread.Name:               runtime.FactoryAdapter(fts, podtopologyspread.New),
		nodeunschedulable.Name:               nodeunschedulable.New,
		noderesources.Name:                   runtime.FactoryAdapter(fts, noderesources.NewFit),
		noderesources.BalancedAllocationName: runtime.FactoryAdapter(fts, noderesources.NewBalancedAllocation),
		volumebinding.Name:                   runtime.FactoryAdapter(fts, volumebinding.New),
		volumerestrictions.Name:              runtime.FactoryAdapter(fts, volumerestrictions.New),
		volumezone.Name:                      volumezone.New,
		nodevolumelimits.CSIName:             runtime.FactoryAdapter(fts, nodevolumelimits.NewCSI),
		nodevolumelimits.EBSName:             runtime.FactoryAdapter(fts, nodevolumelimits.NewEBS),
		nodevolumelimits.GCEPDName:           runtime.FactoryAdapter(fts, nodevolumelimits.NewGCEPD),
		nodevolumelimits.AzureDiskName:       runtime.FactoryAdapter(fts, nodevolumelimits.NewAzureDisk),
		nodevolumelimits.CinderName:          runtime.FactoryAdapter(fts, nodevolumelimits.NewCinder),
		interpodaffinity.Name:                interpodaffinity.New,
		queuesort.Name:                       queuesort.New,
		defaultbinder.Name:                   defaultbinder.New,
		defaultpreemption.Name:               runtime.FactoryAdapter(fts, defaultpreemption.New),
	}
}

func (r Registry) Register(name string, factory PluginFactory) error {
	if _, ok := r[name]; ok {
		return fmt.Errorf("a plugin named %v already exists", name)
	}
	r[name] = factory
	return nil
}

func (r Registry) Merge(in Registry) error {
	for name, factory := range in {
		if err := r.Register(name, factory); err != nil {
			return err
		}
	}
	return nil
}

```

函数 NewInTreeRegistry 返回一个 registry，这个 registry 包含了所有内置插件对象的创建方法。Merge 函数将 NewInTreeRegistry 返回的 registry 和 options.frameworkOutOfTreeRegistry 做合并，那么 options.frameworkOutOfTreeRegistry 是什么呢？很明显，options.frameworkOutOfTreeRegistry 就是我们自定义的插件 registry。



options.frameworkOutOfTreeRegistry 是通过 NewSchedulerCommand 函数的入参进行初始化的，如下代码：

```go

//cmd/kube-scheduler/scheduler.go

func main() {
	command := app.NewSchedulerCommand()
	code := cli.Run(command)
	os.Exit(code)
}
// cmd/kube-scheduler/app/server.go

func NewSchedulerCommand(registryOptions ...Option) *cobra.Command {
	opts := options.NewOptions()

	cmd := &cobra.Command{
        
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, opts, registryOptions...)
		},
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), args)
				}
			}
			return nil
		},
	}
    ...
}
```

从代码中可以看到，NewSchedulerCommand 是在 kube-scheduler 的 main 函数中被执行的，但默认是没有任何入参的，那么我们想要使用自定义的插件是不是只要给这个函数传入合适的参数即可？

没错，正是如此，你只需要在你的 go 项目路径下创建一个 go 文件，该文件包含下面内容，你就可以编译出一个属于你自己的 scheduler

其中 NewSchedulerCommand 函数的入参是 Option 类型的对象，而且可以传多个，表示你可以传入多个自定义的插件对象，Option 类型如下

```go
//cmd/kube-scheduler/app/server.go

type Option func(runtime.Registry) error
```

所以我们要实现自己的插件，就需要实现一个方法，这个方法返回一个 framework.Plugin 类型的对象，而 framework.Plugin 是一个接口类型，那么这个对象需要实现 Name() 方法。至于这个自定义插件想要在哪个（或哪些）扩展点插入，你只需要实现对应扩展点的接口即可，即通过接口组合实现。例如，你要在 Filter 扩展点插入自定义插件，你需要实现下面的接口



```go
type FilterPlugin interface {
	Plugin
	Filter(ctx context.Context, state *CycleState, pod *v1.Pod, nodeInfo *NodeInfo) *Status
}
```

即实现 Plugin 接口和 Filter 方法，实现 Plugin 接口很简单，只需要实现 Name 方法即可，Filter 就是插件执行时候的执行方法。

我们以创建一个 Filter 扩展点插件为例总结一下当我们需要开发一个自定义插件时的流程



```go

package my_plugin

// 1. 定义一个插件结构体
type MyPlugin struct{}

// 2. 实现 Plugin 插件，即实现 Name 方法
func (pl *MyPlugin) Name() {
    return "myPluginName"
}

// 3. 实现 Filter 函数
func (pl *MyPlugin) Filter(ctx context.Context, state *CycleState, pod *v1.Pod, nodeInfo *NodeInfo) *framework.Status {
    你的代码逻辑
}

// 4. 实现 New 函数，返回该自定义插件对象，类似下面代码
func New(_ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &MyPlugin{}, nil
}
```



下面就可以在你的 scheduler main 函数中引用上面创建的插件了，如下：

```go

package main

import (
	"os"

	"k8s.io/component-base/cli"
	_ "k8s.io/component-base/logs/json/register" // for JSON log format registration
	_ "k8s.io/component-base/metrics/prometheus/clientgo"
	_ "k8s.io/component-base/metrics/prometheus/version" // for version metric registration
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
	"my_plugin"
)

func main() {
    myPlugin := app.WithPlugin("myPluginName", my_plugin.New)
	command := app.NewSchedulerCommand(myPlugin)
	code := cli.Run(command)
	os.Exit(code)
}

```
