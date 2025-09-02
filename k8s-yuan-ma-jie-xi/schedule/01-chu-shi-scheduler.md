---
description: scheduler
---

# 01-初识 Scheduler

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*hGZ6MqU9c1gClWgR.png" alt=""><figcaption></figcaption></figure>

调度程序负责决定 Pod 在集群中的部署位置。这听起来像是一项简单的工作，但实际上相当复杂！让我们从基础开始！



当您使用 kubectl 提交 deployment 时，API 服务器会收到请求，并将资源存储在 etcd 中。那&#x4E48;_&#x8C01;创建了 Pod？_

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*fhb_XLY2TFSJNG6k.png" alt=""><figcaption></figcaption></figure>

~~创建 Pod 是调度程序的工作~~，是一个常见的误解。**相反，controller manager 创建它们（以及关联的 ReplicaSet）。**

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*8qbnBTBaEFntnJII.png" alt=""><figcaption></figcaption></figure>

此时，Pod 在 etcd 中存储为“Pending”，并且未分配给任何节点。它们也被添加到调度程序的队列中，准备分配节点。

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*h89BoOCvu2GkyWJx.png" alt=""><figcaption></figcaption></figure>

调度程序通过两个阶段处理 Pod：

* Scheduling phase (what node should I choose?).
* Binding phase (let’s write to the database that this pod belongs to that node).

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*L9AHwwjBrKqfnHuK.png" alt=""><figcaption></figcaption></figure>

Scheduler阶段分为两部分：

1. Filters relevant nodes (using a list of functions called predicates) 过滤
2. Ranks the remaining nodes (using a list of functions called priorities) 打分

让我们看一个例子：

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*SqXqOGmk7r89ep09.png" alt=""><figcaption></figcaption></figure>

考虑以下集群，其中包含带和不带 GPU 的节点，此外，一些节点已经满负荷运行。

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*T6Cmo9nNQAXmkYiJ.png" alt=""><figcaption></figcaption></figure>

您想要部署一个需要一些 GPU 的 Pod。您将 pod 提交到集群，并将其添加到调度程序队列中。调度程序丢弃所有没有 GPU 的节点（过滤阶段）。

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*YCNxG1LLZ0-lo6ly.png" alt=""><figcaption></figcaption></figure>

接下来，调度程序对剩余节点进行评分。在此示例中，资源利用率越高的节点得分较低。我们的调度程序最终选择空节点1。

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*XjTpM0F7Q_8we7aN.png" alt=""><figcaption></figcaption></figure>

过滤器有哪些示例？

* `NodeUnschedulable` 防止 Pod 登陆标记为不可调度的节点。
* `VolumeBinding` 检查节点是否可以绑定请求的卷。

默认过滤阶段有 13 个谓词：

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*4LFbMk2HyDpm7nDx.png" alt=""><figcaption></figcaption></figure>

以下是一些打分示例：

* `ImageLocality` 优先选择已经在本地下载了容器镜像的节点。
* `NodeResourcesBalancedAllocation` 更喜欢未充分利用的节点。

有 13 个函数来决定如何对节点进行评分和排名：

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*h1nAAmrCgNH80Pnz.png" alt=""><figcaption></figcaption></figure>

## 你如何影响调度者的决定？

* `nodeSelector`
* Node affinity
* Pod affinity/anti-affinity
* Taints and tolerations
* Topology constraints
* Scheduler profiles

`nodeSelector` 是最直接的机制，您为节点分配标签并将该标签添加到 Pod，Pod 只能部署在具有该标签的节点上。

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*SIaG_Ijq8Ku3zLkF.png" alt=""><figcaption></figcaption></figure>

Node affinity 通过更灵活的接口扩展了nodeSelector。您仍然可以告诉调度程序 Pod 应该部署在哪里，但您也可以有软约束和硬约束。

<figure><img src="https://miro.medium.com/v2/resize:fit:1400/0*9DFuEmePkddfcr3_.png" alt=""><figcaption></figcaption></figure>

通过 Pod 亲和性/反亲和性，您可以要求调度程序将 pod 放置在特定 pod 旁边。

通过 taints 和 tolerations，pod 会被 tainted ，并且节点会排斥（或容忍）pod。

这与节点亲和力 node affinity 类似，但有一个显着的区别：通过节点亲和力，Pod 会被节点吸引。污点则相反——它们允许节点排斥 pod。



<figure><img src="../../../.gitbook/assets/image (2) (1) (1) (1) (1) (1) (1) (1) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

此外，Tolerations 可以通过三种效果来排斥 Pod：

* NoExecute
* NoSchedule
* PreferNoSchedule

[https://kubernetes.io/zh-cn/docs/concepts/scheduling-eviction/taint-and-toleration/](https://kubernetes.io/zh-cn/docs/concepts/scheduling-eviction/taint-and-toleration/)
