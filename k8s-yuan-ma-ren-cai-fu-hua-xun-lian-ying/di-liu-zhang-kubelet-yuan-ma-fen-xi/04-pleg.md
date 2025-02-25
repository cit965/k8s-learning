# 04-pleg

## 1. PLEG 的基本概念

PLEG 是 Kubelet 中的一个重要组件,主要负责:

* 监控容器状态变化

2\. 生成容器生命周期事件

* 更新 Pod 缓存状态

## 2. PLEG 的两种实现

Kubernetes 提供了两种 PLEG 实现:

* GenericPLEG (传统实现):

```go
type GenericPLEG struct {
    runtime Runtime
    cache Cache 
    eventChannel chan *PodLifecycleEvent
    podRecords podRecords  // 记录 Pod 状态
    relistPeriod time.Duration  // 重新列举周期
}
```

* EventedPLEG (基于事件的新实现):

```go
type EventedPLEG struct {
    runtime Runtime
    cache Cache
    eventChannel chan *PodLifecycleEvent 
    runtimeService RuntimeService // CRI 服务
}
```

## 3. 工作原理

**GenericPLEG 工作流程:**

* 定期重列举(Relist):

```go
func (g *GenericPLEG) Relist() {
    // 1. 获取所有 Pod 列表
    podList, err := g.runtime.GetPods(ctx, true)
    
    // 2. 对比状态变化
    for pid := range g.podRecords {
        // 生成事件
        events := generateEvents(...)
        
        // 发送事件
        g.eventChannel <- events
    }
    
    // 3. 更新缓存
    g.cache.Set(...)
}
```

* 状态对比:

```go
func generateEvents(oldState, newState plegContainerState) []*PodLifecycleEvent {
    switch newState {
    case plegContainerRunning:
        return ContainerStarted
    case plegContainerExited:
        return ContainerDied
    }
}
```

**EventedPLEG 工作流程:**

* 基于 CRI 事件流:

```go
func (e *EventedPLEG) watchEventsChannel() {
    // 1. 订阅 CRI 事件
    containerEventsResponseCh := make(chan *runtimeapi.ContainerEventResponse)
    
    // 2. 处理事件
    for event := range containerEventsResponseCh {
        e.processCRIEvent(event)
    }
}
```

* 事件处理:

```go
func (e *EventedPLEG) processCRIEvent(event *ContainerEventResponse) {
    switch event.Type {
    case ContainerStarted:
        e.sendPodLifecycleEvent(ContainerStarted)
    case ContainerStopped:
        e.sendPodLifecycleEvent(ContainerDied)
    }
}
```

## 4. 主要优势

* EventedPLEG 相比 GenericPLEG 的优势:
* 实时性更好(基于事件)
* 资源消耗更低(无需轮询)
* 延迟更低
* 可靠性保证:// 失败重试机制if numAttempts >= e.eventedPlegMaxStreamRetries {    // 降级到 GenericPLEG    e.Stop()    e.genericPleg.Start()}

## 5. 事件类型

PLEG 生成的主要事件类型:

```go
const (
    ContainerStarted  = "ContainerStarted"
    ContainerDied    = "ContainerDied"
    ContainerRemoved = "ContainerRemoved"
    ContainerChanged = "ContainerChanged"
)
```

## 6. 缓存更新机制

```go
func (g *GenericPLEG) updateCache(pod *Pod) {
    // 1. 获取最新状态
    status := g.runtime.GetPodStatus()
    
    // 2. 保留 Pod IP
    status.IPs = g.getPodIPs()
    
    // 3. 更新缓存
    g.cache.Set(pod.ID, status)
}
```

这种设计确保了:

* 容器状态变化的可靠检测
* 事件的及时传递
* Pod 状态的一致性维护
