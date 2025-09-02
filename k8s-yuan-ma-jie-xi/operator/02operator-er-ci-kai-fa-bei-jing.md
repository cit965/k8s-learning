---
description: operator
---

# 02-operator 二次开发背景

## Custom resources <a href="#custom-resources" id="custom-resources"></a>

**资源（Resource）** 是 [Kubernetes API](https://kubernetes.io/zh-cn/docs/concepts/overview/kubernetes-api/) 中的一个端点， 其中存储的是某个类别的 [API 对象](https://kubernetes.io/zh-cn/docs/concepts/overview/working-with-objects/#kubernetes-objects)的一个集合。 例如内置的 **Pod** 资源包含一组 Pod 对象。

**定制资源（Custom Resource）** 是对 Kubernetes API 的扩展，不一定在默认的 Kubernetes 安装中就可用。定制资源所代表的是对特定 Kubernetes 安装的一种定制。 不过，很多 Kubernetes 核心功能现在都用定制资源来实现，这使得 Kubernetes 更加模块化。

定制资源可以通过动态注册的方式在运行中的集群内或出现或消失，集群管理员可以独立于集群更新定制资源。 一旦某定制资源被安装，用户可以使用 [kubectl](https://kubernetes.io/zh-cn/docs/reference/kubectl/) 来创建和访问其中的对象，就像他们为 **Pod** 这种内置资源所做的一样。

## Custom resources  Definition <a href="#custom-resources" id="custom-resources"></a>

CRD（Custom Resource Definition，自定义资源定义）提供了定义自定义资源的机制，它可以指定自定义资源的结构、属性、类型等信息。

CRD 有点类似于在数据库中定义的表结构，而 CR 则是基于 CRD 模板创建的一个实例。

有的同学可能会问，既然 CRD 与数据库中定义的表结构类似，那为什么我们不直接使用像 MySQL 这类的数据库来对资源进行抽象和设计呢？

主要原因是 CRD 能够完美地和 Kubernetes 生态进行协同工作，能够使用很多 k8s 特性如准入控制，版本管理，级联删除等。

例如，我们可以通过 kubectl 工具对自定义资源进行 CRUD 等操作，还可以使用 Kubernetes 的认证、权限、审计等机制。这样可以避免重复造轮子，让开发同学将精力集中在业务逻辑的实现上，提高效率并且降低成本。

## **Custom Controller**

就定制资源本身而言，它只能用来存取结构化的数据。 当你将定制资源与**定制控制器（Custom Controller）** 结合时， 定制资源就能够提供真正的**声明式 API（Declarative API）**。Custom Controller 会监听 CR 变化，并做出相应处理。

Kubernetes 本身就自带了一堆 Controller，Master 节点上的三大核心组件之一：Controller Manager，其实就是一堆 Controller 的集合

<figure><img src="../../.gitbook/assets/image (18).png" alt=""><figcaption></figcaption></figure>

声明式 API[ ](https://kubernetes.io/zh-cn/docs/concepts/extend-kubernetes/api-extension/custom-resources/#declarative-apis) vs  命令式 API\



### 声明式 API

* 你的 API 包含相对而言为数不多的、尺寸较小的对象（资源）。
* 对象更新操作频率较低。
* 通常需要人来读取或写入对象。
* 对象的主要操作是 CRUD 风格的（创建、读取、更新和删除）。
* 不需要跨对象的事务支持：API 对象代表的是期望状态而非确切实际状态。

### 命令式 API

* 客户端发出“做这个操作”的指令，之后在该操作结束时获得同步响应。
* 客户端发出“做这个操作”的指令，并获得一个操作 ID，之后需要检查一个 Operation（操作） 对象来判断请求是否成功完成。
* 你会将你的 API 类比为远程过程调用（Remote Procedure Call，RPC）。
* 直接存储大量数据；例如每个对象几 kB，或者存储上千个对象。
* 需要较高的访问带宽（长期保持每秒数十个请求）。
* 存储有应用来处理的最终用户数据（如图片、个人标识信息（PII）等）或者其他大规模数据。
* 在对象上执行的常规操作并非 CRUD 风格。
* API 不太容易用对象来建模。

## 添加定制资源[ ](https://kubernetes.io/zh-cn/docs/concepts/extend-kubernetes/api-extension/custom-resources/#adding-custom-resources) <a href="#adding-custom-resources" id="adding-custom-resources"></a>

Kubernetes 提供了两种方式供你向集群中添加定制资源：

* CRD 相对简单，创建 CRD 可以不必编程。
* [API 聚合](https://kubernetes.io/zh-cn/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/)需要编程， 但支持对 API 行为进行更多的控制，例如数据如何存储以及在不同 API 版本间如何转换等。

Kubernetes 提供这两种选项以满足不同用户的需求，这样就既不会牺牲易用性也不会牺牲灵活性。

聚合 API 指的是一些下位的 API 服务器，运行在主 API 服务器后面；主 API 服务器以代理的方式工作。这种组织形式称作 [API 聚合（API Aggregation，AA）](https://kubernetes.io/zh-cn/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/) 。 对用户而言，看起来仅仅是 Kubernetes API 被扩展了。

CRD 允许用户创建新的资源类别同时又不必添加新的 API 服务器。 使用 CRD 时，你并不需要理解 API 聚合。

无论以哪种方式安装定制资源，新的资源都会被当做定制资源，以便与内置的 Kubernetes 资源（如 Pods）相区分。

<figure><img src="../../.gitbook/assets/截屏2024-06-23 14.34.09.png" alt=""><figcaption></figcaption></figure>

## Operator 诞生的背景 <a href="#operator-dan-sheng-de-bei-jing" id="operator-dan-sheng-de-bei-jing"></a>

<figure><img src="../../.gitbook/assets/image (13).png" alt=""><figcaption></figcaption></figure>

kubernetes 无法做到真正意义的开箱即用的，它与传统的 PaaS 平台不同，它仅仅只提供核心的基础设施功能，但是还无法满足用户的最终需求，这里用户主要指业务开发和业务运维， 比如说业务开发需要 CI/CD 工具实现 Devops 的功能，原生 kubernetes 是不提供支持的，但是我们可以通过`tekton`这一个第三方工具实现 DevOps 相关功能， 这也正是 kubernetes 区别传统PaaS平台的真正强大之处，其提供完善的扩展机制以及基于此而发展出来的海量的第三方工具和丰富的生态。



`Operator pattern`首先由 CoreOS 提出，通过结合 CRD 和 custom controller 将特定应用的运维知识转换为代码，实现应用运维的自动化和智能化。 Operator 允许 kubernetes 来管理复杂的，有状态的分布式应用程序，并由 kubernetes 对其进行自动化管理，例如，etcd operator 能够创建并管理一组 etcd 集群， 定制化的 controller 组件了解这些资源，知道如何维护这些特定的应用。



随着 kubernetes 的功能越来越复杂，其需要管理的资源在高速增长，对应的 API 和 controller 的数量也愈发无法控制， kubernetes 变得很臃肿，很多不必要的 API 和功能将出现在每次安装的集群中。

<figure><img src="../../.gitbook/assets/image (14).png" alt=""><figcaption></figcaption></figure>

为了解决这个问题，CRD 应运而生，CRD 由 TPR(Third Part Resource v1.2 版本引入)演化而来，v1.7 进入 beta，v1.8 进入稳定， 通过 CRD，kubernetes 可以动态的添加并管理资源。CRD 解决了结构化数据存储的问题，Controller 则用来跟踪这些资源， 保证资源的状态满足期望值。`CRD+Controller=decalartive API`，声明式 API 设计是 kubernetes 重要的设计思想， 该设计保证能够动态扩展 kubernetes API，这种模式也正是 Operator pattern。

kubernetes 本身也在通过 CRD 添加新功能，我们有什么理由不使用呢？

### 使用场景总结及举例 <a href="#shi-yong-chang-jing-zong-jie-ji-ju-li" id="shi-yong-chang-jing-zong-jie-ji-ju-li"></a>

CRD+custom controller 已经被广泛地使用，按使用场景可划分为以下两种：

1. 通用型 controller: 这种场景和 kubernetes 内置的`apps controller`类似，主要解决特定通用类型应用的管理
2. Operator: 该场景用于解决一个特定应用的自动化管理

通用型 controller 往往是 kubernetes 平台侧用户，如各大云厂商和 kubernetes 服务提供商，Operator 则是各种软件服务提供商， 他们设计时面向单一应用，很多开源的应用的 operator 可以在 operator hub 中获取。我列举一些示例供大家参考：

### 通用型 Controller <a href="#tong-yong-xing-controller" id="tong-yong-xing-controller"></a>

1. 阿里的 cafedeploymentcontroller 解决金融场景下分布式应用特殊需求。
2. oam-kubernetes-runtime 实现了 Application Model (OAM)，以系统可持续的方式拓展 kubernetes

### Operator <a href="#operator" id="operator"></a>

1. etcd operator
2. Prometheus operator

通用型Controller与kubernetes自带的几个controller类似，旨在解决一些通用的应用模型，而Operator则更加面向单个特定应用， 这两者没有本质的区别。

### 如何开发 CRD <a href="#ru-he-kai-fa-crd" id="ru-he-kai-fa-crd"></a>

作为kubernetes开发者，如何开发 CRD+Custom cntroller 呢？其实官方提供示例项目sample-controller 供开发者参考，开发流程大致有以下几个过程：

1. 初始化项目结构（可根据 sample controller 修改）
2. 定义 CRD
3. 生成代码
4. 初始化 controller
5. 实现 controller 具体逻辑

\
其中步骤 2，5 是核心业务逻辑，其余步骤完全可以通过自动生成的方式省略，到目前，社区有两个成熟的脚手架工具用于简化开发，一个是有 kube-sig 维护的 kubebuilder, 另一个是由 redhat 维护的 operator-sdk，这两个工具都是基于 controller-runtime 项目而实现，用户可自行选择，笔者用的是 kubebuilder。 使用 kubebuilder 能够帮助我们节省以下工作：

<figure><img src="../../.gitbook/assets/image (15).png" alt=""><figcaption></figcaption></figure>

如果你想要快速构建 CRD 和 Custom controller，脚手架工具是个不错的选择，如果是学习目的，建议结合 sample-controller 和 kubernetes controller 相关 源码。

### kubebuilder 详解 <a href="#kubebuilder-xiang-jie" id="kubebuilder-xiang-jie"></a>

kubebuilder 是开发自定义控制器的脚手架工具，能给我们搭建好控制器的整个骨架，我们只需要专心编写控制（调谐）逻辑即可，大大方便了控制器的开发流程。

kubebuilder 是一个帮助开发者快速开发 kubernetes API 的脚手架命令行工具，其依赖库 controller-tools 和 controller-runtime， controller-runtime 简化 kubernetes controller 的开发，并且对 kubernetes 的几个常用库进行了二次封装， 以简化开发者使用。controller-tool 主要功能是代码生成。下图是使用 kubebuilder 的工作流程图：

<figure><img src="../../.gitbook/assets/image (16).png" alt=""><figcaption></figcaption></figure>

文章后面会结合一个简单示例来介绍开发流程。

> kubebuilder 有非常良好的文档，包括一个从零开始的示例，您应该以文档为主。

### 使用 kubebuilder 开发一个 CRD <a href="#shi-yong-kubebuilder-kai-fa-yi-ge-crd" id="shi-yong-kubebuilder-kai-fa-yi-ge-crd"></a>

本次示例创建一个通用的`Application`资源，Application 包含一个子资源 Deployment 以及一个 Count 资源， 每当 Application 进行被重新协调**Reconcil**，Count 会进行自增。

#### 前提（你需要提前了解的） <a href="#qian-ti-ni-xu-yao-ti-qian-liao-jie-de" id="qian-ti-ni-xu-yao-ti-qian-liao-jie-de"></a>

1. Golang 开发者，kubernetes 大量使用`Code Generate`这一功能来自动生成重复性代码
2. 阅读 kubernetes controller 的代码
3. 阅读 kubebuilder 的文档

#### 开发步骤及主要代码展示 <a href="#kai-fa-bu-zhou-ji-zhu-yao-dai-ma-zhan-shi" id="kai-fa-bu-zhou-ji-zhu-yao-dai-ma-zhan-shi"></a>

首先，根据你的开发环境安装 kubebuilder 工具，mac 下通过 homebrew 安装命令如下：

```bash
➜  ~ brew install kubebuilder
➜  ~ kubebuilder version
Version: version.Version{KubeBuilderVersion:"unknown", KubernetesVendor:"unknown", GitCommit:"$Format:%H$", BuildDate:"1970-01-01T00:00:00Z", GoOs:"unknown", GoArch:"unknown"}
```

安装完毕后，首先创建项目目录`custom-controller`并使用`go mod`初始化项目

```bash
➜  cd custom-controllers
➜  ls
➜  go mod init controllers.happyhack.io
```

接着，使用 kubebuilder 初始化项目，生成相关文件和目录，并创建 CRD 资源\`

```bash
# 使用kubebuilder初始化项目
➜  custom-controllers kubebuilder init --domain controller.daocloud.io --license apache2 --owner "Holder"
# 创建CRD资源
➜  custom-controllers kubebuilder create api --group controller --version v1 --kind Application
Create Resource [y/n]
y
Create Controller [y/n]
y
Writing scaffold for you to edit...
api/v1/application_types.go
controllers/application_controller.go
Running make:
$ make
/Users/donggang/Documents/Code/golang/bin/controller-gen object:headerFile="hack/huilerplate.go.txt" paths="./..."
go fmt ./...
go vet ./...
go build -o bin/manager main.go

```

到目前，该项目就已经能够运行了，不过我们需要添加我们自己业务代码，主要包括修改 CRD 定义和添加 controller 逻辑两部分。 首先修改 API 资源定义即 CRD 定义，Application 包含一个 Deployment，我们可以参考 kubernetes Deployment 与 Pod 这两个类型之间的关系设计 Application 和 Deployment， Deployment 通过字段`spec.template`来描述如何创建 Pod，`DeploymentTemplateSpec`描述了该如何创建 Deployment，

```go
// PodTemplateSpec describes the data a deployment should have when created from a template
type DeploymentTemplateSpec struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the pod.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec v12.DeploymentSpec `json:"spec,omitempty"`
}
```

API 描述定义完成了，接下来需要我们来进行具体业务逻辑实现了，编写具体 controller 实现，首先我们简单梳理 controller 的主要逻辑

* 当一个 application 被创建时，需要创建对应的 deployment，当 application 被删除或更新时，对应 Deployment 也需要被删除或更新
* 当 application 对应的子资源 deployment 被其他客户端删除或更新时，controller 需要重建或恢复它
* 最后一步更新 application 的 status，这里即 count 加 1

我们在方法`func(r *ApplicationReconciler) Reconcile(req ctrl.Request)(ctrl.Result,error)`实现相关逻辑， 当然当业务逻辑比较复杂时，可以拆分为多个方法。

```go
func (r *ApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("application", req.NamespacedName)

	var app controllerv1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		log.Error(err, "unable to fetch app")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	selector, err := metav1.LabelSelectorAsSelector(app.Spec.Selector)
	if err != nil {
		log.Error(err, "unable to convert label selector")
		return ctrl.Result{}, err
	}

	var deploys v12.DeploymentList
	if err := r.List(ctx, &deploys, client.InNamespace(req.Namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		if errors.IsNotFound(err) {
			deploy, err := r.constructDeploymentForApplication(&app)
			if err != nil {
				log.Error(err, "unable to construct deployment")
				return ctrl.Result{
					RequeueAfter: time.Second * 1,
				}, err
			}
			if err = r.Create(ctx, deploy); err != nil {
				return ctrl.Result{RequeueAfter: time.Second * 1}, err
			}
		}
	}
	...
}

```

完成`Reconcile`方法后，我们可以修改config目录的示例yaml，来进行本地测试了。

### 官方开发自定义 Controller 的指导 <a href="#guan-fang-kai-fa-zi-ding-yi-controller-de-zhi-dao" id="guan-fang-kai-fa-zi-ding-yi-controller-de-zhi-dao"></a>

kubernetes开箱自带了多个controller，这些controller在我们开发时具有非常重要的参考价值，同时社区也总结了的 controller 开发所需要遵循十一条原则， 但是请大家结合实际场景灵活运用这些原则：

<figure><img src="../../.gitbook/assets/image (17).png" alt=""><figcaption></figcaption></figure>

### 总结及展望 <a href="#zong-jie-ji-zhan-wang" id="zong-jie-ji-zhan-wang"></a>

本文简单介绍了 CRD 以及如何使用脚手架工具 kubebuilder 帮助我们开发自定义 controller，当然这个 controller 示例的逻辑比较简单， 在实际场景中，我们会遇到很多的挑战，比如controller 的逻辑会比较复杂、需要通过多个controller等。作为kubernetes开发者， Controller开发是一项必不可少的技能。
