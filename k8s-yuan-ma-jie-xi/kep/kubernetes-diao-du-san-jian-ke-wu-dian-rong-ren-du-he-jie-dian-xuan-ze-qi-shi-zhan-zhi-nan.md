# Kubernetes 调度三剑客：污点、容忍度和节点选择器实战指南

> 作为运维工程师，你是否遇到过这些问题：GPU 节点被普通应用占用？数据库 Pod 跑到了低配节点？维护节点时 Pod 乱跑？

### 一、背景：一个真实的生产事故

上周五晚上 8 点，我接到告警：AI 训练任务失败了。排查后发现，价值 10 万的 GPU 节点上跑满了普通的 Web 应用，AI 任务根本抢不到资源，这个事故让公司损失了一整天的训练时间。痛定思痛，我决定好好学习下节点资源隔离！

### 二、核心概念：用酒店管理来理解

#### 把 K8s 集群想象成一家连锁酒店

```
┌─────────────────────────────────────────┐
│          K8s 大酒店集团                  │
├─────────────────────────────────────────┤
│ 标准房间（普通节点）                  │
│    - 欢迎所有客人（Pod）                 |
├─────────────────────────────────────────┤
│ VIP 套房（GPU 节点）                  │
│    - 只接待 VIP 客人（AI Pod）           │
│    - 需要出示 VIP 卡（容忍度）           │
├─────────────────────────────────────────┤
│ 维修房间（维护节点）                  │
│    - 暂停接待新客人                      │
│    - 现有客人请尽快离开                  │
└─────────────────────────────────────────┘
```

#### 三个核心概念对照表

| K8s 概念                  | 酒店类比    | 作用对象 | 实际作用            |
| ----------------------- | ------- | ---- | --------------- |
| **Taint（污点）**           | 房间门禁    | Node | 节点说："我拒绝某些 Pod" |
| **Toleration（容忍度）**     | VIP 通行证 | Pod  | Pod 说："我能进特殊节点" |
| **NodeSelector（节点选择器）** | 客人偏好    | Pod  | Pod 说："我只去特定节点" |

### 三、污点（Taints）：节点的三种"脾气"

#### NoSchedule：坚决不接待

```bash
# 给 GPU 节点打上"专用"标记
kubectl taint nodes gpu-node-1 hardware=gpu:NoSchedule
```

**效果**：新 Pod 绝对不会调度到这个节点（除非有容忍度）

**类比**：会员制俱乐部，非会员禁止入内

**适用场景**：

* GPU 节点专用
* 数据库专用节点
* 生产环境隔离

#### PreferNoSchedule：最好别来

```bash
# 高内存节点，建议小任务别来
kubectl taint nodes mem-node-1 memory=high:PreferNoSchedule
```

**效果**：调度器会尽量避开，但资源紧张时还是会调度过来

**类比**：高峰期地铁，不建议老人乘坐（但可以坐）

**适用场景**：

* 资源优先级控制
* 软性隔离需求

#### NoExecute：现在就走

```bash
# 节点要维护，让现有 Pod 离开
kubectl taint nodes node-5 maintenance=true:NoExecute
```

**效果**：

1. 新 Pod 不会调度过来
2. 已有 Pod 会被驱逐（除非有容忍度）

**类比**：餐厅打烊，请现有客人离开

**适用场景**：

* 节点维护
* 紧急故障处理
* 节点下线

### 四、实战案例：从入门到精通

#### 案例 1：GPU 节点专用（入门级）

**场景**：公司有 2 台 GPU 服务器，只想跑 AI 训练任务

**步骤 1：给 GPU 节点打污点**

```bash
# 标记 GPU 节点
kubectl taint nodes gpu-node-1 hardware=gpu:NoSchedule
kubectl taint nodes gpu-node-2 hardware=gpu:NoSchedule

# 同时打上标签（方便选择）
kubectl label nodes gpu-node-1 hardware=gpu
kubectl label nodes gpu-node-2 hardware=gpu

# 验证配置
kubectl describe node gpu-node-1 | grep -A 5 "Taints"
```

**输出示例**：

```
Taints:             hardware=gpu:NoSchedule
```

**步骤 2：创建 AI 训练任务**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: ai-training-job
  labels:
    app: ai-training
spec:
  # 武器1：容忍度（通行证）
  tolerations:
  - key: "hardware"
    operator: "Equal"
    value: "gpu"
    effect: "NoSchedule"
  
  # 武器2：节点选择器（主动选择）
  nodeSelector:
    hardware: gpu
  
  containers:
  - name: trainer
    image: tensorflow/tensorflow:2.13.0-gpu
    resources:
      limits:
        nvidia.com/gpu: 1  # 申请 1 块 GPU
    command: ["python", "train.py"]
```

**步骤 3：创建普通 Web 应用（对比）**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: web-app
spec:
  # 注意：没有容忍度，也没有节点选择器
  containers:
  - name: nginx
    image: nginx:1.25
```

**调度结果**：

```
✅ ai-training-job  → 调度到 gpu-node-1 或 gpu-node-2
❌ web-app         → 只能调度到普通节点（GPU 节点被拒绝）
```

**验证效果**：

```bash
# 查看 Pod 调度情况
kubectl get pods -o wide

# 输出示例
NAME              NODE          STATUS
ai-training-job   gpu-node-1    Running
web-app           worker-1      Running
```

#### 案例 2：数据库节点专用（进阶级）

**场景**：MySQL 需要高性能 SSD 节点，不希望其他应用干扰

**完整配置**

```yaml
# 1. 先给 SSD 节点打标记
# kubectl taint nodes ssd-node-1 workload=database:NoSchedule
# kubectl label nodes ssd-node-1 disk-type=ssd workload=database

---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  serviceName: mysql
  replicas: 3
  selector:
    matchLabels:
      app: mysql
  template:
    metadata:
      labels:
        app: mysql
    spec:
      # 容忍度：允许调度到数据库专用节点
      tolerations:
      - key: "workload"
        operator: "Equal"
        value: "database"
        effect: "NoSchedule"
      
      # 节点选择器：只选择 SSD 节点
      nodeSelector:
        disk-type: ssd
        workload: database
      
      # 亲和性：尽量分散到不同节点（高可用）
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: mysql
              topologyKey: kubernetes.io/hostname
      
      containers:
      - name: mysql
        image: mysql:8.0
        env:
        - name: MYSQL_ROOT_PASSWORD
          value: "your-password"
        resources:
          requests:
            memory: "4Gi"
            cpu: "2"
          limits:
            memory: "8Gi"
            cpu: "4"
        volumeMounts:
        - name: data
          mountPath: /var/lib/mysql
  
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: ssd-storage
      resources:
        requests:
          storage: 100Gi
```

**配置解读**：

```
┌─────────────────────────────────────────┐
│ 三重保障机制                             │
├─────────────────────────────────────────┤
│ 1️⃣ Toleration（容忍度）                 │
│    → 能进入有污点的数据库节点            │
├─────────────────────────────────────────┤
│ 2️⃣ NodeSelector（节点选择器）           │
│    → 只选择 SSD + database 标签的节点    │
├─────────────────────────────────────────┤
│ 3️⃣ PodAntiAffinity（反亲和性）          │
│    → 3 个副本分散到不同节点（高可用）    │
└─────────────────────────────────────────┘
```

#### 案例 3：节点维护（实战级）

**场景**：周末要升级 node-5 的内核，需要平滑驱逐 Pod

**维护流程**

```bash
# 步骤 1：标记节点为维护状态（给 Pod 5 分钟迁移时间）
kubectl taint nodes node-5 maintenance=true:NoExecute

# 步骤 2：禁止新 Pod 调度（双保险）
kubectl cordon node-5

# 步骤 3：观察 Pod 迁移情况
watch kubectl get pods -o wide | grep node-5

# 步骤 4：确认 Pod 全部迁移后，进行维护
# ... 升级内核、重启等操作 ...

# 步骤 5：维护完成，恢复节点
kubectl uncordon node-5
kubectl taint nodes node-5 maintenance:NoExecute-
```

**关键业务 Pod 配置（需要延迟驱逐）**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: payment
  template:
    metadata:
      labels:
        app: payment
    spec:
      # 容忍维护状态，但只能待 30 分钟
      tolerations:
      - key: "maintenance"
        operator: "Equal"
        value: "true"
        effect: "NoExecute"
        tolerationSeconds: 1800  # 30 分钟缓冲时间
      
      containers:
      - name: payment
        image: payment-service:v2.1
        
        # 优雅关闭配置
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sh", "-c", "sleep 15"]  # 等待请求处理完
        
        # 健康检查
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      
      # Pod 优雅关闭时间
      terminationGracePeriodSeconds: 60
```

**时间线**：

```
T+0s   : 打上 maintenance 污点
T+0s   : 普通 Pod 立即开始驱逐
T+0s   : payment-service 继续运行（有容忍度）
T+30m  : payment-service 也开始驱逐
T+31m  : 所有 Pod 已迁移完成
T+31m  : 开始维护节点
```

<br>
