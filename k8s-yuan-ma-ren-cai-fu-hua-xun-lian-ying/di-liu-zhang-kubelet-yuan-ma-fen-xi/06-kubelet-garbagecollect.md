# 06-kubelet garbageCollect

Kubelet 的垃圾回收机制主要包含三个部分：

* 容器垃圾回收（Container GC）
* 镜像垃圾回收（Image GC）
* Pod 垃圾回收（Pod GC）

容器垃圾回收负责清理已终止的容器，确保每个 Pod 只保留特定数量的已终止容器；镜像垃圾回收负责清理未使用的镜像，当磁盘使用率超过阈值时触发清理；Pod 垃圾回收则负责清理已终止的 Pod 及其相关资源。

这三个垃圾回收机制通过周期性运行来维护节点的资源使用，防止资源耗尽，确保节点的稳定运行。每种垃圾回收都有其独立的策略配置，如最小年龄、保留数量、触发阈值等，通过这些配置可以灵活控制垃圾回收的行为。

## 代码调用链

```go
// RunKubelet -> createAndInitKubelet -> StartGarbageCollection

func RunKubelet(ctx context.Context, kubeServer *options.KubeletServer, kubeDeps *kubelet.Dependencies) error {
   
    k, err := createAndInitKubelet(kubeServer,kubeDeps,hostname,hostnameOverridden,nodeName,nodeIPs)

    startKubelet(k, podCfg, &kubeServer.KubeletConfiguration, kubeDeps, kubeServer.EnableServer)
  

    return nil
}

func createAndInitKubelet(kubeServer *options.KubeletServer,
	kubeDeps *kubelet.Dependencies,
	hostname string,
	hostnameOverridden bool,
	nodeName types.NodeName,
	nodeIPs []net.IP) (k kubelet.Bootstrap, err error) {

	k, err = kubelet.NewMainKubelet()

	k.StartGarbageCollection()

	return k, nil
}

func (kl *Kubelet) StartGarbageCollection() {
	// 垃圾回收
}

```

## 关键代码

```go

// StartGarbageCollection 启动垃圾回收线程
func (kl *Kubelet) StartGarbageCollection() {
    // 容器 GC 失败标记，用于控制日志级别
    loggedContainerGCFailure := false
    
    // 1. 启动容器垃圾回收协程
    go wait.Until(func() {
        ctx := context.Background()
        // 执行容器垃圾回收
        if err := kl.containerGC.GarbageCollect(ctx); err != nil {
            // 垃圾回收失败处理
            klog.ErrorS(err, "Container garbage collection failed")
            // 记录失败事件
            kl.recorder.Eventf(kl.nodeRef, v1.EventTypeWarning, 
                events.ContainerGCFailed, err.Error())
            loggedContainerGCFailure = true
        } else {
            // 垃圾回收成功处理
            // 根据之前是否失败决定日志级别
            var vLevel klog.Level = 4
            if loggedContainerGCFailure {
                // 如果之前失败过，第一个成功的 GC 使用更低的日志级别（更容易看到）
                vLevel = 1
                loggedContainerGCFailure = false
            }
            klog.V(vLevel).InfoS("Container garbage collection succeeded")
        }
    }, ContainerGCPeriod, wait.NeverStop)  // 定期执行，永不停止

    // 2. 检查是否需要启动镜像垃圾回收
    // 当 high threshold 设为 100，且 max age 为 0（或 max age 功能没启动）时，不启动镜像 GC
    if kl.kubeletConfiguration.ImageGCHighThresholdPercent == 100 &&
        (!utilfeature.DefaultFeatureGate.Enabled(features.ImageMaximumGCAge) || 
         kl.kubeletConfiguration.ImageMaximumGCAge.Duration == 0) {
        klog.V(2).InfoS("ImageGCHighThresholdPercent is set 100 and " +
            "ImageMaximumGCAge is 0, Disable image GC")
        return
    }

    // 3. 启动镜像垃圾回收协程
    // 记录上一次 GC 是否失败
    prevImageGCFailed := false
    // 记录 GC 开始时间
    beganGC := time.Now()
    
    go wait.Until(func() {
        ctx := context.Background()
        // 执行镜像垃圾回收
        if err := kl.imageManager.GarbageCollect(ctx, beganGC); err != nil {
            if prevImageGCFailed {
                // 连续失败时记录错误并创建事件
                klog.ErrorS(err, "Image garbage collection failed multiple times in a row")
                kl.recorder.Eventf(kl.nodeRef, v1.EventTypeWarning, 
                    events.ImageGCFailed, err.Error())
            } else {
                // 首次失败可能是因为统计信息未初始化完成
                klog.ErrorS(err, "Image garbage collection failed once. " +
                    "Stats initialization may not have completed yet")
            }
            prevImageGCFailed = true
        } else {
            // 垃圾回收成功处理
            var vLevel klog.Level = 4
            if prevImageGCFailed {
                vLevel = 1
                prevImageGCFailed = false
            }
            klog.V(vLevel).InfoS("Image garbage collection succeeded")
        }
    }, ImageGCPeriod, wait.NeverStop)  // 定期执行，永不停止
}
```

## 关键特性

1. 并发执行：

```go
// 容器和镜像垃圾回收并行执行
go containerGC()  // 独立协程
go imageGC()      // 独立协程
```

2. 周期性调度：

```go
// 不同类型的垃圾回收可以有不同的周期
ContainerGCPeriod = 1 * time.Minute
ImageGCPeriod = 5 * time.Minute
```

3. 错误隔离：

```go
// 一个垃圾回收任务的失败不影响其他任务
if err := gc1.GarbageCollect(); err != nil {
    // 只记录错误，继续执行
    klog.ErrorS(err, "GC failed")
}
```

4. 永续运行：

```go
// wait.NeverStop 确保垃圾回收永远运行
wait.Until(..., period, wait.NeverStop)
```
