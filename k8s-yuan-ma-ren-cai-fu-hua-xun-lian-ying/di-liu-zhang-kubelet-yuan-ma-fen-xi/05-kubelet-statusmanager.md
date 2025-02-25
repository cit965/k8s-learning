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
