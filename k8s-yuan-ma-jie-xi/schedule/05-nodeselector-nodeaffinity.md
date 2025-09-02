# 05-nodeSelector,nodeAffinity

进入主题之前，先看看创建pod的大概过程:

<figure><img src="../../../.gitbook/assets/截屏2024-06-27 17.48.50.png" alt=""><figcaption></figcaption></figure>

1. kubectl向apiserver发起创建pod请求，apiserver将创建pod配置写入etcd
2. scheduler收到apiserver有新pod的事件，scheduler根据自身调度算法选择一个合适的节点，并打标记pod=test-b-k8s-node01
3. kubelet收到分配到自己节点的pod，调用[docker](https://cloud.tencent.com/product/tke?from\_column=20065\&from=20065) api创建[容器](https://cloud.tencent.com/product/tke?from\_column=20065\&from=20065)，并获取容器状态汇报给apiserver
4. 执行kubectl get查看，apiserver会再次从etcd查询pod信息

> k8s的各个组件是基于list-watch机制进行交互的，了解了list-watch机制，以及结合上述pod的创建流程，就可以很好的理解各个组件之间的交互。

**何为调度**

> 说白了就是将Pod指派到合适的节点上，以便对应节点上的Kubelet能够运行这些Pod。在k8s中，承担调度工作的组件是kube-scheduler，它也是k8s集群的默认调度器，它在设计上就允许编写一个自定义的调度组件并替换原有的kube-scheduler。所以，如果你足够牛逼，就可以自己开发一个调度器来替换默认的了。调度器通过K8S的监测（Watch）机制来发现集群中新创建且尚未被调度到节点上的Pod，调度器会将所发现的每一个未调度的Pod调度到一个合适的节点上来运行。

* 调度程序会过滤掉任何不满足Pod特定调度需求的节点
* 创建Pod时也可以手动指定一个节点
* 如果没有任何一个节点能满足Pod的资源请求， 那么这个Pod将一直停留在未调度状态直到调度器能够找到合适的Node

**调度流程**

> kube-scheduler给一个Pod做调度选择时包含了两个步骤：过滤、打分。

1. pod开始创建，通知apiserver
2. kube-scheduler在集群中找出所有满足需求的可调度节点（过滤阶段）
3. kube-scheduler根据当前打分规则给这些可调度节点打分（打分阶段）
4. kube-scheduler选择得分最高的节点运行Pod（存在多个得分最高的节点则随机选取）
5. kube-scheduler通知kube-apiserver

**nodeSelector和nodeAffinity**

> 实际工作中，可能会有这样的情况，需要进一步控制Pod被部署到哪个节点。例如，确保某些Pod最终落在具有SSD硬盘的主机上，又需要确保某些pod落在具体部门的主机上运行，这时就可以使用标签选择器来进行选择。

<figure><img src="../../../.gitbook/assets/截屏2024-06-27 17.53.21.png" alt=""><figcaption></figcaption></figure>
