# KEP-3157 allow informers for getting a stream of data instead of chunking.

### &#x20;概括

kube-apiserver 容易出现内存爆炸。这个问题在较大的集群中很明显，少数 LIST 请求可能会导致严重的中断。服务器不受控制和无限制的内存消耗不仅会影响以 HA 模式运行的集群，还会影响共享同一台机器的其他程序。在本 KEP 中，我们提出了解决此问题的方案。

### &#x20;动机

如今，informers 是 LIST 请求的主要来源。 LIST 用于获取一致的数据快照，以构建客户端内存缓存。 LIST 请求的主要问题是不可预测的内存消耗。实际使用情况取决于许多因素，例如页面大小、应用的过滤器（例如标签选择器）、查询参数和单个对象的大小

<figure><img src="../../../.gitbook/assets/image (52).png" alt=""><figcaption></figcaption></figure>

在极端情况下，服务器可以为每个请求分配数百兆字节。为了更好地形象化这个问题，让我们考虑一下上面的图表。它显示了测试期间 API 服务器的内存使用情况。我们可以看到，增加 informers 数量会大大增加服务器的内存消耗。此外，16点40分左右，我们在运行了16个 informers 后失去了服务器。在调查过程中，我们意识到服务器分配了大量内存来处理 LIST 请求。简而言之，它需要从数据库中获取数据，对其进行解码，进行一些转换并为客户端准备最终响应。底线是大约 O(5\*the\_response\_from\_etcd) 的临时内存消耗。优先级和公平性以及 Golang 垃圾回收都无法保护系统免于耗尽内存。

这样的情况是双重危险的。首先，正如我们所见，如果不完全停止已收到请求的 API 服务器，它的速度可能会变慢。其次，内存消耗突然且不受控制的激增可能会给节点本身带来压力。这可能会导致系统崩溃、饥饿，并最终丢失同一节点上运行的其他进程，包括 kubelet。停止 kubelet 会带来严重的问题，因为它会导致工作负载中断和更大的爆炸半径。请注意，在这种情况下，即使是 HA 设置中的集群也会受到影响。

更糟糕的是，在极少数情况下，具有许多 kubelet 的大型集群的恢复以及 pod、秘密、configmap 的通知可能会导致非常昂贵的 LIST 风暴。

### 提议

为了在获取数据列表时降低内存消耗并使其更具可预测性，我们建议使用来自 watch-cache 的流式传输而不是来自 etcd 的分页。主要思想是使用标准 WATCH 请求机制来获取单个对象的流，但将其用于 LIST。这将使我们能够保持内存分配不变。服务器受限于 etcd 中 1.5 MB 对象的最大允许大小（请注意，内存中的同一对象可以更大，甚至是一个数量级）加上一些额外的分配，这将在本节后面进行解释文档。大致想法/计划如下：

* 步骤 1：使用新的查询参数建立 WATCH 请求，而不是 LIST 请求。
* 步骤 2：收到来自 informers 的请求后，计算应返回结果的 RV。它将用于确保监视缓存已看到直到接收到的 RV 为止的对象。此步骤是必要的，可确保我们满足请求的一致性要求。
* 步骤 2a：发送当前存储在内存中的给定资源类型的所有对象。
* 步骤 2b：传播同时可能发生的任何更新，直到监视缓存赶上步骤 2 中收到的最新 RV。
* 步骤 2c：使用给定的 RV 向 informers 发送书签事件。
* 步骤 3：使用步骤 1 中的请求侦听进一步的事件。

### &#x20;设计细节

<figure><img src="../../../.gitbook/assets/image (53).png" alt=""><figcaption></figcaption></figure>

<figure><img src="../../../.gitbook/assets/image (54).png" alt=""><figcaption></figcaption></figure>

详情参考： [https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/3157-watch-list](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/3157-watch-list)
