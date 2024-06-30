# 04-Scheduler 二次开发

## scheduler 两种拓展方式

我们聊了聊Pod是如何被识别出来需要被调度，以及它是怎样被调度到合适的节点上的。在调度过程中，系统会用到一些预设的插件来找到合适的节点。不过，kube-scheduler这个调度器是可以进行个性化扩展的。那么，为什么我们想要扩展它呢？原因可能是预设的插件算法不能满足我们的特定需求。比如，我们可能希望在调度的早期阶段就排除掉带有某些特定标签的节点，或者我们想要根据节点的实际资源使用情况来评分，而不是仅仅看它已经分配了多少资源。在这种情况下，预设的kube-scheduler就不够用了，我们就需要自己开发一个插件来扩展它。

kube-scheduler 扩展的方式主要有下面两种：

1. **使用扩展器（extender）机制**：这种方式允许你开发独立的程序来执行过滤、评分和绑定操作。你可以用任何编程语言来开发，并且这些程序可以运行在任何kube-scheduler能够通过HTTP访问到的地方。这种方式的好处是，它让调度器和扩展程序保持独立，你不需要修改kube-scheduler的代码，只需要增加一些配置文件。但缺点是，调度器需要通过网络调用这些扩展程序，这会消耗网络资源，可能会降低调度的速度，并且对外部扩展的稳定性有很高的依赖。
2. **通过调度框架（Scheduling Framework）扩展**：从Kubernetes v1.15版本开始，引入了一种可插拔的调度框架，让定制调度器变得更加简单。你只需要在调度流程的某个点插入自己的插件，甚至可以关闭掉一些不需要的默认插件。这种方式不需要修改kube-scheduler 的代码，而且你的插件会运行在调度器框架内部，就像内置插件一样，因此不需要进行网络调用，稳定性和效率都更高。就像下面的图示，每个箭头都代表一个可以插入自定义插件的扩展点。

本文就根据第二种模式详细讲讲，开发一个自定义的插件需要哪些步骤：

## schedule framework 拓展方式

<figure><img src="../../../.gitbook/assets/image (9).png" alt=""><figcaption></figcaption></figure>

我们先回顾下几个重要的概念：scheduler, framework, registry ，他们之间的关系我用下图来表示：

<figure><img src="../../../.gitbook/assets/image (10).png" alt=""><figcaption></figcaption></figure>

* **scheduler**

是一个调度器，它实现了整体调度逻辑，如在适当位置执行适当的扩展点，一个 Pod 调度失败了需要做什么处理，记录每一个调度情况暴露出 metric 方便外界对 scheduler 的监控等等。scheduler 是串联整个调度流程的。

* **Profiles**

scheduler 的一个成员，就是如下的一个Map

```go
// pkg/scheduler/profile/profile.go

// Map holds frameworks indexed by scheduler name.
type Map map[string]framework.Framework
```

在Kubernetes中，调度器（scheduler）负责决定哪些Pod应该被部署到哪些节点上。通常，我们使用的是Kubernetes自带的默认调度器，它的名字是"default-scheduler"。这个调度器和它的配置文件我们一般不会去改动。

但是，如果你想使用自己开发的插件来实现一些特殊的调度逻辑，就需要创建一个新的调度器。这个过程就像是给这个调度器起一个名字，比如叫"my-custom-scheduler"，然后在配置文件中告诉Kubernetes，这个新的调度器应该使用哪些插件。

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

在Kubernetes中，我们可以为不同的调度需求定义多个调度配置文件，这些配置文件被称为"profiles"。每个profile都有一个唯一的调度器名称，比如我们定义了两个profile，它们的调度器名称分别是"my-scheduler-1"和"my-scheduler-2"。这样一来，我们的调度配置文件就会有两个键，分别对应这两个调度器名称。

当你想要让某个Pod使用特定的调度配置，你可以在Pod的配置中指定调度器名称。这可以在Pod的配置文件的`.spec.schedulerName`字段中完成，比如：

```yaml
spec:
  schedulerName: my-scheduler-1
```

这样设置后，Pod就会使用名为"my-scheduler-1"的调度器来进行调度。



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

## InTree & outofTree

函数 NewInTreeRegistry 返回一个 registry，这个 registry 包含了所有内置插件对象的创建方法。Merge 函数将 NewInTreeRegistry 返回的 registry 和 options.frameworkOutOfTreeRegistry 做合并，那么 options.frameworkOutOfTreeRegistry 是什么呢？很明显，options.frameworkOutOfTreeRegistry 就是我们自定义的插件 registry。

## 开发 outofTree

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
