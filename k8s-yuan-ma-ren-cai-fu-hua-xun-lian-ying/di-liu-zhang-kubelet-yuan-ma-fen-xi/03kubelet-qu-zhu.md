# 03-kubelet 驱逐

kubelet 监控集群节点的内存、磁盘空间和文件系统的 inode 等资源。 当这些资源中的一个或者多个达到特定的消耗水平， kubelet 可以主动地使节点上一个或者多个 Pod 失效，以回收资源防止饥饿。

在节点压力驱逐期间，kubelet 将所选 Pod 的阶段设置为 `Failed` 并终止 Pod。

## kubelet 驱逐时 Pod 的选择[ ](https://kubernetes.io/zh-cn/docs/concepts/scheduling-eviction/node-pressure-eviction/#kubelet-%E9%A9%B1%E9%80%90%E6%97%B6-pod-%E7%9A%84%E9%80%89%E6%8B%A9) <a href="#kubelet-qu-zhu-shi-pod-de-xuan-ze" id="kubelet-qu-zhu-shi-pod-de-xuan-ze"></a>

如果 kubelet 回收节点级资源的尝试没有使驱逐信号低于条件， 则 kubelet 开始驱逐最终用户 Pod。

kubelet 使用以下参数来确定 Pod 驱逐顺序：

```
1. 资源用量超过 request 的 Pod
2. 按 Pod 优先级（PriorityClass）
3. 相对于 request 的资源超用比例
```

因此，kubelet 按以下顺序排列和驱逐 Pod：

1. 首先考虑资源使用量超过其请求的 `BestEffort` 或 `Burstable` Pod。 这些 Pod 会根据它们的优先级以及它们的资源使用级别超过其请求的程度被逐出。
2. 资源使用量少于请求量的 `Guaranteed` Pod 和 `Burstable` Pod 根据其优先级被最后驱逐。
3. kubelet 不使用 Pod 的 QoS 类来确定驱逐顺序, QoS 主要用于系统 OOM 情况下的进程选择,内核自主行为

## Qos

QoS是`Quality of Service`的缩写，即服务质量。QoS 主要影响 OOM Killer 行为，不直接影响 kubelet 驱逐决策。

### pod QoS级别 <a href="#podqos-ji-bie" id="podqos-ji-bie"></a>

QoS主要分为`Guaranteed`、`Burstable`和`Best-Effort`三个级别，优先级从高到低。

怎么决定某个pod属于哪个QoS分类呢？根据pod yaml中的cpu和内存资源定义决定。

### **Guaranteed**

`Guaranteed` Pod 具有最严格的资源限制，并且最不可能面临驱逐。 在这些 Pod 超过其自身的限制或者没有可以从 Node 抢占的低优先级 Pod 之前， 这些 Pod 保证不会被杀死。这些 Pod 不可以获得超出其指定 limit 的资源。这些 Pod 也可以使用 [`static`](https://kubernetes.io/zh-cn/docs/tasks/administer-cluster/cpu-management-policies/#static-policy) CPU 管理策略来使用独占的 CPU。

**判据**

同时满足以下情形的pod属于Guaranteed级别：

* Pod 中的每个容器必须有内存 limit 和内存 request。
* 对于 Pod 中的每个容器，内存 limit 必须等于内存 request。
* Pod 中的每个容器必须有 CPU limit 和 CPU request。
* 对于 Pod 中的每个容器，CPU limit 必须等于 CPU request。

```yaml
containers:
  - name: test-a
    resources:
      limits:
        cpu: 10m
        memory: 1Gi
      requests:
        cpu: 10m
        memory: 1Gi
  - name: test-b
    resources:
      limits:
        cpu: 100m
        memory: 100Mi
      requests:
        cpu: 100m
        memory: 100Mi
```

### Burstable <a href="#burstable" id="burstable"></a>

`Burstable` Pod 有一些基于 request 的资源下限保证，但不需要特定的 limit。 如果未指定 limit，则默认为其 limit 等于 Node 容量，这允许 Pod 在资源可用时灵活地增加其资源。 在由于 Node 资源压力导致 Pod 被驱逐的情况下，只有在所有 `BestEffort` Pod 被驱逐后 这些 Pod 才会被驱逐。因为 `Burstable` Pod 可以包括没有资源 limit 或资源 request 的容器， 所以 `Burstable` Pod 可以尝试使用任意数量的节点资源。

**判据**

Pod 被赋予 `Burstable` QoS 类的几个判据：

* Pod 不满足针对 QoS 类 `Guaranteed` 的判据。
* Pod 中至少一个容器有内存或 CPU 的 request 或 limit。

```yaml
containers:
  - name: test-a
...
  - name: test-b
    resources:
      limits:
        memory: 200Mi
      requests:
        memory: 100Mi
```

### BestEffort <a href="#besteffort" id="besteffort"></a>

`BestEffort` QoS 类中的 Pod 可以使用未专门分配给其他 QoS 类中的 Pod 的节点资源。 例如若你有一个节点有 16 核 CPU 可供 kubelet 使用，并且你将 4 核 CPU 分配给一个 `Guaranteed` Pod， 那么 `BestEffort` QoS 类中的 Pod 可以尝试任意使用剩余的 12 核 CPU。

如果节点遇到资源压力，kubelet 将优先驱逐 `BestEffort` Pod。

**判据**

如果 Pod 不满足 `Guaranteed` 或 `Burstable` 的判据，则它的 QoS 类为 `BestEffort`。 换言之，只有当 Pod 中的所有容器没有内存 limit 或内存 request，也没有 CPU limit 或 CPU request 时，Pod 才是 `BestEffort`。Pod 中的容器可以请求（除 CPU 或内存之外的） 其他资源并且仍然被归类为 `BestEffort`

```yaml
...
containers:
  - name: test-a
...
  - name: test-b
...
```

