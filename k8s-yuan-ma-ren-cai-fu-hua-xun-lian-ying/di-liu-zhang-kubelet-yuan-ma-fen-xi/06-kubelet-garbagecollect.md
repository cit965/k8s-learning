# 06-kubelet garbageCollect

Kubelet 的垃圾回收机制主要包含两个部分：

* 容器垃圾回收（Container GC）
* 镜像垃圾回收（Image GC）

容器垃圾回收负责清理已终止的容器，确保每个 Pod 只保留特定数量的已终止容器；镜像垃圾回收负责清理未使用的镜像，当磁盘使用率超过阈值时触发清理。

垃圾回收机制通过周期性运行来维护节点的资源使用，防止资源耗尽，确保节点的稳定运行。每种垃圾回收都有其独立的策略配置，如最小年龄、保留数量、触发阈值等，通过这些配置可以灵活控制垃圾回收的行为。

## 代码调用链

GarbageCollect 是在 kubelet 对象初始化完成后启动的，在 createAndInitKubelet 方法中首先调用 kubelet.NewMainKubelet 初始化了 kubelet 对象，随后调用 k.StartGarbageCollection 启动了 GarbageCollect。

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

## 关键代码 - StartGarbageCollection

容器和镜像垃圾回收并行执行 ，不同类型的垃圾回收可以有不同的周期，一个垃圾回收任务的失败不影响其他任务，只记录错误，继续执行。

* go containerGC()  // 独立协程 ,容器垃圾回收
* go imageGC()  // 独立协程 ,镜像垃圾回收

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

## 关键代码 - containerGC.GarbageCollect(ctx)

```go
// GarbageCollect 根据指定的容器垃圾回收策略清理死亡容器
// 注意：垃圾回收策略不适用于 sandbox。sandbox 只有在未就绪且不包含容器时才会被删除
//
// 参数：
//   - ctx: 上下文
//   - gcPolicy: 垃圾回收策略
//   - allSourcesReady: 所有数据源是否就绪
//   - evictNonDeletedPods: 是否清理未删除的 Pod
//
// 垃圾回收步骤：
// 1. 获取可清理的容器（非活动且创建时间超过 gcPolicy.MinAge）
// 2. 根据 gcPolicy.MaxPerPodContainer 限制清理每个 Pod 最老的死亡容器
// 3. 根据 gcPolicy.MaxContainers 限制清理最老的死亡容器
// 4. 获取可清理的 sandbox（未就绪且不包含容器）
// 5. 清理符合条件的 sandbox
func (cgc *containerGC) GarbageCollect(
    ctx context.Context,
    gcPolicy kubecontainer.GCPolicy,
    allSourcesReady bool,
    evictNonDeletedPods bool,
) error {
    // 创建追踪 span
    ctx, otelSpan := cgc.tracer.Start(ctx, "Containers/GarbageCollect")
    defer otelSpan.End()
    
    // 收集所有错误
    errors := []error{}

    // 1. 清理可回收的容器(不在 running 并且创建时间大于 MinAge 的容器)
    if err := cgc.evictContainers(
        ctx, 
        gcPolicy, 
        allSourcesReady, 
        evictNonDeletedPods,
    ); err != nil {
        errors = append(errors, err)
    }

    // 2. 清理不包含容器的 sandbox
    if err := cgc.evictSandboxes(
        ctx, 
        evictNonDeletedPods,
    ); err != nil {
        errors = append(errors, err)
    }

    // 3. 清理 Pod 的日志目录
    if err := cgc.evictPodLogsDirectories(
        ctx, 
        allSourcesReady,
    ); err != nil {
        errors = append(errors, err)
    }

    // 返回聚合的错误
    return utilerrors.NewAggregate(errors)
}
```

## 关键代码 - imageManager.GarbageCollect(ctx)

```go
func (im *realImageGCManager) GarbageCollect(ctx context.Context, beganGC time.Time) error {
    ctx, otelSpan := im.tracer.Start(ctx, "Images/GarbageCollect")
    defer otelSpan.End()

    freeTime := time.Now()
    images, err := im.imagesInEvictionOrder(ctx, freeTime)
    if err != nil {
       return err
    }

    images, err = im.freeOldImages(ctx, images, freeTime, beganGC)
    if err != nil {
       return err
    }

    // Get disk usage on disk holding images.
    // 获取磁盘空间使用量
    fsStats, _, err := im.statsProvider.ImageFsStats(ctx)
    if err != nil {
       return err
    }

    var capacity, available int64
    if fsStats.CapacityBytes != nil {
       capacity = int64(*fsStats.CapacityBytes)
    }
    if fsStats.AvailableBytes != nil {
       available = int64(*fsStats.AvailableBytes)
    }

    if available > capacity {
       klog.InfoS("Availability is larger than capacity", "available", available, "capacity", capacity)
       available = capacity
    }

    // Check valid capacity.
    if capacity == 0 {
       err := goerrors.New("invalid capacity 0 on image filesystem")
       im.recorder.Eventf(im.nodeRef, v1.EventTypeWarning, events.InvalidDiskCapacity, err.Error())
       return err
    }

    // If over the max threshold, free enough to place us at the lower threshold.
    // 计算当前使用率，如果超过最高阈值，那么尝试释放到最低阈值
    usagePercent := 100 - int(available*100/capacity)
    if usagePercent >= im.policy.HighThresholdPercent {
       // 估算期望释放的总量
       amountToFree := capacity*int64(100-im.policy.LowThresholdPercent)/100 - available
       klog.InfoS("Disk usage on image filesystem is over the high threshold, trying to free bytes down to the low threshold", "usage", usagePercent, "highThreshold", im.policy.HighThresholdPercent, "amountToFree", amountToFree, "lowThreshold", im.policy.LowThresholdPercent)
       // 进行释放
       freed, err := im.freeSpace(ctx, amountToFree, freeTime, images)
       if err != nil {
          return err
       }
       // 如果实际释放量低于期望释放量，那么会打日志并纪录事件
       if freed < amountToFree {
          err := fmt.Errorf("Failed to garbage collect required amount of images. Attempted to free %d bytes, but only found %d bytes eligible to free.", amountToFree, freed)
          im.recorder.Eventf(im.nodeRef, v1.EventTypeWarning, events.FreeDiskSpaceFailed, err.Error())
          return err
       }
    }

    return nil
}
```

举个例子: 我的磁盘总共 100G，然后配置了

* LowThresholdPercent 设置 20
* HighThresholdPercent 设置 80

当前我镜像占用了 90G 的空间，那么这次的 `usagePercent` 为 90，高于配置的 `HighThresholdPercent`，需要执行回收逻辑。期望释放 70G 的空间，来满足最低阈值 `LowThresholdPercent`。如果实际释放的量不足 70G，会有事件纪录。

关键函数有两个:

* `im.statsProvider.ImageFsStats()` 是如何计算使用量的
* `im.freeSpace` 磁盘空间释放逻辑

首先我们来看 `ImageFsStats` 函数。这个的具体实现依 `kubeDeps.useLegacyCadvisorStats` 而定:

* `true` 使用 cadvisor 的 API
* `false` 使用 Container Runtime 的返回的 Image Path 然后自己算的

这里对 CRI 的实现进行展开，`ImageFsStats` 函数的定义如下：

```go
// pkg/kubelet/stats/cri_stats_provider.go:389
// ImageFsStats returns the stats of the image filesystem.
func (p *criStatsProvider) ImageFsStats(ctx context.Context) (imageFsRet *statsapi.FsStats, containerFsRet *statsapi.FsStats, errRet error) {
    resp, err := p.imageService.ImageFsInfo(ctx)
    if err != nil {
       return nil, nil, err
    }

    // CRI may return the stats of multiple image filesystems but we only
    // return the first one.
    //
    // TODO(yguo0905): Support returning stats of multiple image filesystems.
    if len(resp.GetImageFilesystems()) == 0 {
       return nil, nil, fmt.Errorf("imageFs information is unavailable")
    }
    fs := resp.GetImageFilesystems()[0]
    imageFsRet = &statsapi.FsStats{
       Time:      metav1.NewTime(time.Unix(0, fs.Timestamp)),
       UsedBytes: &fs.UsedBytes.Value,
    }
    if fs.InodesUsed != nil {
       imageFsRet.InodesUsed = &fs.InodesUsed.Value
    }
    imageFsInfo, err := p.getFsInfo(fs.GetFsId())
    if err != nil {
       return nil, nil, fmt.Errorf("get filesystem info: %w", err)
    }
    if imageFsInfo != nil {
       // The image filesystem id is unknown to the local node or there's
       // an error on retrieving the stats. In these cases, we omit those
       // stats and return the best-effort partial result. See
       // https://github.com/kubernetes/heapster/issues/1793.
       imageFsRet.AvailableBytes = &imageFsInfo.Available
       imageFsRet.CapacityBytes = &imageFsInfo.Capacity
       imageFsRet.InodesFree = imageFsInfo.InodesFree
       imageFsRet.Inodes = imageFsInfo.Inodes
    }
    // TODO: For CRI Stats Provider we don't support separate disks yet.
    return imageFsRet, imageFsRet, nil
}
```

**磁盘使用量**是通过 `ImageFsInfo` 函数获取的，这个会最终请求到 Container Runtime。比如 Docker 的就会调用 `/info` 这个路径然后拿 JSON 中的 `DockerRootDir` 这个字段，然后通过 Golang std 的 `file path.Walk` 去遍历计算这个目录的大小。关于 Docker API 可以通过 `curl` 调用试试， `curl --unix-socket /var/run/docker.sock http://127.0.0.1/info | jq` ，`DockerRootDir` 一般为路径 `/var/lib/docker`。

**磁盘的总量**是通过 cadvisor 的 API 来获取的挂载点的文件系统信息然后提取的

```go
// pkg/kubelet/stats/cri_stats_provider.go:450
// getFsInfo returns the information of the filesystem with the specified
// fsID. If any error occurs, this function logs the error and returns
// nil.
func (p *criStatsProvider) getFsInfo(fsID *runtimeapi.FilesystemIdentifier) (*cadvisorapiv2.FsInfo, error) {
    if fsID == nil {
       klog.V(2).InfoS("Failed to get filesystem info: fsID is nil")
       return nil, nil
    }
    mountpoint := fsID.GetMountpoint()
    // 这里会调用 cadvisor 的 API
    fsInfo, err := p.cadvisor.GetDirFsInfo(mountpoint)
    if err != nil {
       msg := "Failed to get the info of the filesystem with mountpoint"
       if errors.Is(err, cadvisorfs.ErrNoSuchDevice) ||
          errors.Is(err, cadvisorfs.ErrDeviceNotInPartitionsMap) ||
          errors.Is(err, cadvisormemory.ErrDataNotFound) {
          klog.V(2).InfoS(msg, "mountpoint", mountpoint, "err", err)
       } else {
          klog.ErrorS(err, msg, "mountpoint", mountpoint)
          return nil, fmt.Errorf("%s: %w", msg, err)
       }
       return nil, nil
    }
    return &fsInfo, nil
}
```

最后我们来看一下释放的逻辑:

```go
// pkg/kubelet/images/image_gc_manager.go:447
// Tries to free bytesToFree worth of images on the disk.
//
// Returns the number of bytes free and an error if any occurred. The number of
// bytes freed is always returned.
// Note that error may be nil and the number of bytes free may be less
// than bytesToFree.
func (im *realImageGCManager) freeSpace(ctx context.Context, bytesToFree int64, freeTime time.Time, images []evictionInfo) (int64, error) {
    // Delete unused images until we've freed up enough space.
    var deletionErrors []error
 
    // 记录已释放的空间
    spaceFreed := int64(0)
    // 遍历所有镜像进行评估
    for _, image := range images {
       klog.V(5).InfoS("Evaluating image ID for possible garbage collection based on disk usage", "imageID", image.id, "runtimeHandler", image.imageRecord.runtimeHandlerUsedToPullImage)
       // Images that are currently in used were given a newer lastUsed.
       // 检察镜像是否最近被使用过
       // 如果镜像在 freeTime 之后被使用过，跳过该镜像
       if image.lastUsed.Equal(freeTime) || image.lastUsed.After(freeTime) {
          klog.V(5).InfoS("Image ID was used too recently, not eligible for garbage collection", "imageID", image.id, "lastUsed", image.lastUsed, "freeTime", freeTime)
          continue
       }

       // Avoid garbage collect the image if the image is not old enough.
       // In such a case, the image may have just been pulled down, and will be used by a container right away.
       // 检查镜像 age
       // 如果镜像太新，跳过该镜像，这是避免删除刚刚拉取的镜像
       if freeTime.Sub(image.firstDetected) < im.policy.MinAge {
          klog.V(5).InfoS("Image ID's age is less than the policy's minAge, not eligible for garbage collection", "imageID", image.id, "age", freeTime.Sub(image.firstDetected), "minAge", im.policy.MinAge)
          continue
       }
       // 尝试删除镜像
       if err := im.freeImage(ctx, image, ImageGarbageCollectedTotalReasonSpace); err != nil {
          deletionErrors = append(deletionErrors, err)
          continue
       }
       // 更新已经释放的空间
       spaceFreed += image.size
   
       // 如果我们已经释放足够多的容量满足了期望值，那么中断，保证镜像的使用量不会小于 LowThresholdPercent
       if spaceFreed >= bytesToFree {
          break
       }
    }
    // 处理删除过程中发生的错误
    if len(deletionErrors) > 0 {
       return spaceFreed, fmt.Errorf("wanted to free %d bytes, but freed %d bytes space with errors in image deletion: %v", bytesToFree, spaceFreed, errors.NewAggregate(deletionErrors))
    }
    // 返回成功释放空间大小
    return spaceFreed, nil
}
```
