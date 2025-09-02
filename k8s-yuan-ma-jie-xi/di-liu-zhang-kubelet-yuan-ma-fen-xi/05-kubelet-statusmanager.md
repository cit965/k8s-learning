# 05-kubelet statusManager

## 基本概念

StatusManager 是 Kubelet 中负责管理和同步 Pod 状态的组件。它维护了一个本地缓存来存储 Pod 的最新状态，通过版本号机制确保状态更新的顺序性。当 Pod 状态发生变化时（如容器启动、终止等），StatusManager 会更新本地缓存并触发同步流程。它会定期（默认10秒）或在状态变更时立即与 API Server 同步，确保集群状态的一致性。同时，它还处理特殊情况如静态 Pod 的状态同步（通过 Mirror Pod）、状态转换的合法性检查、以及终止状态的正确处理。这个组件是 Kubelet 保证 Pod 状态准确性和实时性的关键机制。

## 启动

StatusManager 会启动一个永久运行的同步循环。这个循环通过两种方式触发状态同步：一是通过 podStatusChannel 接收到的即时状态更新信号，二是通过每 10 秒触发一次的定时全量同步。这种机制既保证了状态更新的实时性（即时触发），又确保了状态的最终一致性（定时同步）。

```go
// Run starts the kubelet reacting to config updates
func (kl *Kubelet) Run(updates <-chan kubetypes.PodUpdate) {

    // Start volume manager
    go kl.volumeManager.Run(ctx, kl.sourcesReady)

    // Start component sync loops.
    kl.statusManager.Start()

    // Start the pod lifecycle event generator.
    kl.pleg.Start()

    kl.syncLoop(ctx, updates, kl)
}

func (m *manager) Start() {
	// 如果开启了 InPlacePodVerticalScaling 功能门禁，就需要给 state 字段赋值，以便记录资源分配状态
	// Initialize m.state to no-op state checkpoint manager
	m.state = state.NewNoopStateCheckpoint()
	// Create pod allocation checkpoint manager even if client is nil so as to allow local get/set of AllocatedResources & Resize
	if utilfeature.DefaultFeatureGate.Enabled(features.InPlacePodVerticalScaling) {
		stateImpl, err := state.NewStateCheckpoint(m.stateFileDirectory, podStatusManagerStateFile)
		if err != nil {
			// This is a crictical, non-recoverable failure.
			klog.ErrorS(err, "Could not initialize pod allocation checkpoint manager, please drain node and remove policy state file")
			panic(err)
		}
		m.state = stateImpl
	}

	// Don't start the status manager if we don't have a client. This will happen
	// on the master, where the kubelet is responsible for bootstrapping the pods
	// of the master components.
	if m.kubeClient == nil {
		klog.InfoS("Kubernetes client is nil, not starting status manager")
		return
	}

	klog.InfoS("Starting to sync pod status with apiserver")



    	// 创建一个每 10 秒触发一次的定时器通道
	//nolint:staticcheck // SA1015 Ticker can leak since this is only called once and doesn't handle termination.
	syncTicker := time.NewTicker(syncPeriod).C

	// syncPod and syncBatch share the same go routine to avoid sync races.
    	// 启动永久运行的同步协程
    	// syncPod 和 syncBatch 共享同一个协程以避免同步竞争
	go wait.Forever(func() {
		for {
			select {
			case <-m.podStatusChannel:
				klog.V(4).InfoS("Syncing updated statuses")
				m.syncBatch(false)
			case <-syncTicker:
				klog.V(4).InfoS("Syncing all statuses")
				m.syncBatch(true)
			}
		}
	}, 0)
}

```

## SyncBatch

`syncBatch` 是定期将 statusManager 缓存 podStatuses 中的数据同步到 apiserver 的方法，主要逻辑为：

1、调用 `m.podManager.GetUIDTranslations` 从 podManager 中获取 mirrorPod uid 与 staticPod uid 的对应关系；

2、从  apiStatusVersions 中清理已经不存在的 pod，遍历 apiStatusVersions，检查 podStatuses 以及 mirrorToPod 中是否存在该对应的 pod，若不存在则从 apiStatusVersions 中删除；

3、遍历 podStatuses，首先调用 `needsUpdate` 检查 pod 的状态是否与 apiStatusVersions 中的一致，然后调用 `needsReconcile` 检查 pod 的状态是否与 podManager 中的一致，若不一致则将需要同步的 pod 加入到 updatedStatuses 列表中；

4、遍历 updatedStatuses 列表，调用 `m.syncPod` 方法同步状态；

syncBatch 主要是将 statusManage cache 中的数据与 apiStatusVersions 和 podManager 中的数据进行对比是否一致，若不一致则以 statusManage cache 中的数据为准同步至 apiserver。

```go
// syncBatch 将 Pod 状态与 apiserver 同步
// 参数 all: 是否执行全量同步
// 返回值: 尝试同步的数量（用于测试）
func (m *manager) syncBatch(all bool) int {
    // podSync 定义了需要同步的 Pod 信息
    type podSync struct {
        podUID    types.UID               // Pod 的 UID
        statusUID kubetypes.MirrorPodUID  // 状态对应的 UID（对于静态 Pod 是 Mirror Pod 的 UID）
        status    versionedPodStatus      // 带版本的 Pod 状态
    }

    // 存储需要更新的状态列表
    var updatedStatuses []podSync
    
    // 获取 Pod UID 的转换映射
    // podToMirror: 普通Pod/静态Pod UID -> Mirror Pod UID
    // mirrorToPod: Mirror Pod UID -> 静态Pod UID
    podToMirror, mirrorToPod := m.podManager.GetUIDTranslations()

    // 关键部分：使用读锁保护并发访问
    func() {
        m.podStatusesLock.RLock()
        defer m.podStatusesLock.RUnlock()

        // 清理孤立的版本记录
        // 仅在全量同步时执行
        if all {
            for uid := range m.apiStatusVersions {
                // 检查 Pod 和 Mirror Pod 是否都不存在
                _, hasPod := m.podStatuses[types.UID(uid)]
                _, hasMirror := mirrorToPod[uid]
                if !hasPod && !hasMirror {
                    // 删除不再需要的版本记录
                    delete(m.apiStatusVersions, uid)
                }
            }
        }

        // 遍历所有 Pod 状态，决定哪些需要更新
        for uid, status := range m.podStatuses {
            // 获取用于 API 服务器的状态 UID
            // 对于静态 Pod，需要使用其 Mirror Pod 的 UID
            uidOfStatus := kubetypes.MirrorPodUID(uid)
            
            // 处理静态 Pod 的特殊情况
            if mirrorUID, ok := podToMirror[kubetypes.ResolvedPodUID(uid)]; ok {
                // 如果静态 Pod 没有对应的 Mirror Pod，跳过处理
                if mirrorUID == "" {
                    klog.V(5).InfoS("Static pod does not have a corresponding mirror pod; skipping",
                        "podUID", uid,
                        "pod", klog.KRef(status.podNamespace, status.podName))
                    continue
                }
                uidOfStatus = mirrorUID
            }

            // 增量更新模式
            if !all {
                // 只有新版本的状态才需要更新
                if m.apiStatusVersions[uidOfStatus] >= status.version {
                    continue
                }
                updatedStatuses = append(updatedStatuses, podSync{uid, uidOfStatus, status})
                continue
            }

            // 全量更新模式：检查是否需要更新或重新同步
            // 1. 检查是否需要更新（版本不匹配或其他原因）
            if m.needsUpdate(types.UID(uidOfStatus), status) {
                updatedStatuses = append(updatedStatuses, podSync{uid, uidOfStatus, status})
            } else if m.needsReconcile(uid, status.status) {
                // 2. 检查是否需要重新同步（状态不一致）
                // 删除版本记录强制更新
                delete(m.apiStatusVersions, uidOfStatus)
                updatedStatuses = append(updatedStatuses, podSync{uid, uidOfStatus, status})
            }
        }
    }()

    // 执行实际的状态同步
    for _, update := range updatedStatuses {
        klog.V(5).InfoS("Sync pod status", 
            "podUID", update.podUID, 
            "statusUID", update.statusUID, 
            "version", update.status.version)
        m.syncPod(update.podUID, update.status)
    }

    return len(updatedStatuses)
}
```

## SyncPod

`syncPod` 是用来同步 pod 最新状态至 apiserver 的方法，主要逻辑为：

1、从 apiserver 获取 pod 的 oldStatus；

2、检查 pod `oldStatus` 与 `currentStatus` 的 uid 是否相等，若不相等则说明 pod 被重建过；

3、调用 `statusutil.PatchPodStatus` 同步 pod 最新的 status 至 apiserver，并将返回的 pod 作为 newPod；

4、检查 newPod 是否处于 terminated 状态，若处于 terminated 状态则调用 apiserver 接口进行删除并从 cache 中清除，删除后 pod 会进行重建；

```go
// syncPod 将给定的状态与 API 服务器同步
// 参数：
//   - uid: Pod 的 UID
//   - status: 带版本的 Pod 状态
// 注意：调用者不能持有状态锁
func (m *manager) syncPod(uid types.UID, status versionedPodStatus) {
    // 1. 从 API 服务器获取最新的 Pod 信息
    pod, err := m.kubeClient.CoreV1().Pods(status.podNamespace).Get(
        context.TODO(), 
        status.podName, 
        metav1.GetOptions{},
    )
    
    // 处理 Pod 不存在的情况
    if errors.IsNotFound(err) {
        klog.V(3).InfoS("Pod does not exist on the server",
            "podUID", uid,
            "pod", klog.KRef(status.podNamespace, status.podName))
        // Pod 已被删除，状态将在 RemoveOrphanedStatuses 中清理
        // 这里直接忽略更新
        return
    }
    
    // 处理其他错误
    if err != nil {
        klog.InfoS("Failed to get status for pod",
            "podUID", uid,
            "pod", klog.KRef(status.podNamespace, status.podName),
            "err", err)
        return
    }

    // 2. 处理 Pod 重建场景
    // 获取转换后的 Pod UID（处理静态 Pod 的情况）
    translatedUID := m.podManager.TranslatePodUID(pod.UID)
    // 如果 Pod 被删除后重建，UID 会改变，此时需要跳过状态更新
    if len(translatedUID) > 0 && translatedUID != kubetypes.ResolvedPodUID(uid) {
        klog.V(2).InfoS("Pod was deleted and then recreated, skipping status update",
            "pod", klog.KObj(pod),
            "oldPodUID", uid,
            "podUID", translatedUID)
        // 删除旧的状态记录
        m.deletePodStatus(uid)
        return
    }

    // 3. 合并 Pod 状态
    // 将本地状态与 API 服务器状态合并，考虑容器运行状态
    mergedStatus := mergePodStatus(
        pod.Status, 
        status.status,
        m.podDeletionSafety.PodCouldHaveRunningContainers(pod),
    )

    // 4. 更新 API 服务器中的状态
    // 使用 patch 操作更新状态，避免覆盖其他字段
    newPod, patchBytes, unchanged, err := statusutil.PatchPodStatus(
        context.TODO(),
        m.kubeClient,
        pod.Namespace,
        pod.Name,
        pod.UID,
        pod.Status,
        mergedStatus,
    )
    klog.V(3).InfoS("Patch status for pod", 
        "pod", klog.KObj(pod), 
        "podUID", uid, 
        "patch", string(patchBytes))

    // 处理更新错误
    if err != nil {
        klog.InfoS("Failed to update status for pod", 
            "pod", klog.KObj(pod), 
            "err", err)
        return
    }

    // 5. 处理更新结果
    if unchanged {
        // 状态未发生变化
        klog.V(3).InfoS("Status for pod is up-to-date", 
            "pod", klog.KObj(pod), 
            "statusVersion", status.version)
    } else {
        // 状态已更新
        klog.V(3).InfoS("Status for pod updated successfully", 
            "pod", klog.KObj(pod), 
            "statusVersion", status.version, 
            "status", mergedStatus)
        pod = newPod
        // 记录启动延迟相关指标
        m.podStartupLatencyHelper.RecordStatusUpdated(pod)
    }

    // 6. 记录状态更新耗时
    if status.at.IsZero() {
        klog.V(3).InfoS("Pod had no status time set", 
            "pod", klog.KObj(pod), 
            "podUID", uid, 
            "version", status.version)
    } else {
        duration := time.Since(status.at).Truncate(time.Millisecond)
        metrics.PodStatusSyncDuration.Observe(duration.Seconds())
    }

    // 7. 更新本地版本记录
    m.apiStatusVersions[kubetypes.MirrorPodUID(pod.UID)] = status.version

    // 8. 处理 Pod 删除
    // 注意：不处理 Mirror Pod 的优雅删除
    if m.canBeDeleted(pod, status.status, status.podIsFinished) {
        // 设置删除选项，立即删除
        deleteOptions := metav1.DeleteOptions{
            GracePeriodSeconds: new(int64),
            // 使用 Pod UID 作为删除前提条件，防止删除同名的新 Pod
            Preconditions: metav1.NewUIDPreconditions(string(pod.UID)),
        }
        
        // 从 API 服务器删除 Pod
        err = m.kubeClient.CoreV1().Pods(pod.Namespace).Delete(
            context.TODO(), 
            pod.Name, 
            deleteOptions,
        )
        if err != nil {
            klog.InfoS("Failed to delete status for pod", 
                "pod", klog.KObj(pod), 
                "err", err)
            return
        }
        
        klog.V(3).InfoS("Pod fully terminated and removed from etcd", 
            "pod", klog.KObj(pod))
        // 清理本地状态
        m.deletePodStatus(uid)
    }
}
```

## State

status 目录下还有个 state 文件夹，这个文件夹下的代码是为了支持 InPlacePodVerticalScaling 功能，支持 Pod 的动态资源调整和确保 Kubelet 重启后的状态恢复。

<figure><img src="../../.gitbook/assets/image (83).png" alt=""><figcaption></figcaption></figure>
