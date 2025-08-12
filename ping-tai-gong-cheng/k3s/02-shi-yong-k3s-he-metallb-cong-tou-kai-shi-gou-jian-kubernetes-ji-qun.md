# 02-使用 K3s 和 MetalLB 从头开始​​构建 Kubernetes 集群

Kubernetes 是一个由 Google 最初开发的开源项目。它用于容器编排，简单来说就是管理多台计算机上的容器.

我计划搭建一个家庭实验室，用于托管我的个人项目和其他需要的应用程序。我想模拟真实的生产环境，并通过管理 Kubernetes 集群、容器编排和网络配置来积累经验。此外，我可以在一个不怕破坏的环境中进行实验。目前我没有硬件可供使用，因此我正在使用计算机上的虚拟机来搭建一切。一旦我有了所需的设备来构建一个小型数据中心，我会将从这个项目中学到的知识应用到“真实”的集群中.

在这篇文章中，我将解释我如何从零开始使用 K3S 在虚拟机上搭建一个 Kubernetes 集群.

## 搭建集群

在运行 Kubernetes 集群之前，你需要有节点或服务器来安装 Kubernetes 软件。我的桌面运行在 Windows 上，所以我使用 [Hyper-V](https://learn.microsoft.com/en-us/virtualization/hyper-v-on-windows/about/) 来创建虚拟机以托管集群。我的集群有四台虚拟机；一个控制节点和三个工作节点，它们都在同一个网络中运行 K3S。虚拟机位于一个 /24 网络中，DHCP 和 DNS 由 dnsmasq 管理。下面的图片展示了我当前的集群配置：

<figure><img src="../../.gitbook/assets/image (41).png" alt=""><figcaption></figcaption></figure>

对于服务器的基本操作系统，我选择了我偏好的操作系统；Ubuntu Server。每台服务器都有 4GB 的内存、24GB 的磁盘和一个静态 IP。

## 关键组件和配置

在准备好服务器后，是时候安装 Kubernetes 了。一个 Kubernetes 集群由控制平面和一个或多个工作节点组成。控制平面是一组管理 Kubernetes 节点的进程。它负责控制运行哪些应用程序以及它们使用的镜像。它管理调度、网络和集群的整体状态。工作节点用来运行由集群管理的应用程序.

我在集群中使用了 [K3S](https://k3s.io/)，因为它是一个轻量级、精简版的 Kubernetes，非常适合在资源受限的设备上运行，比如我计划在家庭实验室集群中使用的 Raspberry PIs。K3S 可以通过一个 shell 脚本安装：

## **在主节点上安装 K3S**

```bash
curl -sfL https://get.k3s.io | sh -
```

上述命令将 K3s 安装为 `systemd` 服务，并安装了许多其他有用的集群和容器管理工具，如 `kubectl`、`crictl`、`ctr`、`k3s-killall.sh` 和 `k3s-uninstall.sh`。安装脚本会在 `/etc/rancher/k3s/k3s.yaml` 中创建一个 `kubeconfig` 文件，该文件包含有关集群的信息。`kubectl` 使用此文件和集群通信.

## 在工作节点上安装 K3S

使用上述脚本安装 K3S 后，主节点将成为一个单节点集群。要将其他节点加入集群，我在每个节点上运行了以下命令，并设置了两个环境变量；`K3S_URL` 和 `K3S_TOKEN`，指向主节点。`K3S_URL` 是主节点的主机名或 IP 地址。`K3S_TOKEN` 用于工作节点与主节点上运行的控制平面进行身份验证，该令牌确认节点有权加入集群。此令牌存储在主节点的 `/var/lib/rancher/k3s/server/node-token` 文件中.

```bash
# Point the worker nodes to the leader node and supply its token
curl -sfL https://get.k3s.io | K3S_URL=https://myserver:6443 K3S_TOKEN=mynodetoken sh -

# Example
curl -sfL https://get.k3s.io | K3S_URL=https://the-hub:6443 K3S_TOKEN=WB9M05DmGQh6fARsvM4rEWiz0eoNNTNWb9QMGwZYpW4= sh -
```

在主节点上，注意节点的 IP 地址或主机名，并从 `/var/lib/rancher/k3s/server/node-token` 中复制其 K3S\_TOKEN.

如果你运行了防火墙，如 IP Tables 或 UFW，你需要打开 Kubernetes 用于与节点通信的 TCP 端口 6443，并添加规则以允许来自其创建的网络（**10.42.0.0/16** 和 **10.43.0.0/16**）的流量.

```bash
ufw allow 6443/tcp #apiserver
ufw allow from 10.42.0.0/16 to any #pods
ufw allow from 10.43.0.0/16 to any #services
```

在工作节点中添加 K3S 环境变量会使 K3s 以代理或工作模式启动。这种设置类似于你如何设置 Docker Swarm 集群，你可以在服务器节点上生成一个令牌，并使用主节点的令牌将工作节点添加或加入到集群中.

安装完一切后，我的四节点集群就运行起来了。下面的图片展示了运行 `kubectl get nodes` 命令的输出，该命令显示了 Kubernetes 集群的健康和状态。有三个工作节点和一个主节点。

<figure><img src="../../.gitbook/assets/image (42).png" alt=""><figcaption></figcaption></figure>

## 部署应用程序

目前集群中只有一个应用程序在运行，但将应用程序部署到 Kubernetes 集群的步骤是类似的。你需要&#x20;

1\) 将应用程序容器化，将其推送到像 Docker Hub 这样的镜像仓库

2\) 为应用程序创建 Kubernetes 清单

3\) 设置网络、存储和配置

4\) 将配置应用到集群.

### 容器化应用程序

我构建了一个 Transformers API 的镜像并将其推送到 [Docker Hub](https://hub.docker.com/r/vndlovu/transformers-app)。Kubernetes 在部署应用程序时会从 Docker Hub 拉取此镜像。它不处理镜像的构建，因为它的重点是在集群中运行容器。构建容器镜像是另一个需要使用像 Docker 这样的工具来处理的问题.

### 创建 Kubernetes 清单

Kubernetes 中的清单是 YAML 或 JSON 配置文件，让你可以定义应用程序在 Kubernetes 集群中的期望状态。清单类似于 Docker Compose 文件，你可以用它们来指定你想要部署的应用程序的镜像、副本数量以及你的应用程序应该运行在哪些节点上。我在这里讨论的清单是一些摘录，用来解释一些概念。

下面是一个来自 Transformers API 的清单文件示例：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: transformers-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: transformers
  template:
    metadata:
      labels:
        app: transformers
    spec:
      containers:
      - name: transformers-web
        image: vndlovu/transformers-app
        ports:
          - containerPort: 8000
        env:
          - name: POSTGRES_USER
            valueFrom:
              secretKeyRef:
                name: postgres-secret
                key: POSTGRES_USER
          - name: POSTGRES_PASSWORD
            valueFrom:
              secretKeyRef:
                name: postgres-secret
                key: POSTGRES_PASSWORD
          - name: POSTGRES_DB
            value: transformers_db

          - name: NODE_NAME
            valueFrom:
              fieldRef:
                fieldPath: spec.nodename
```

此清单指示 Kubernetes 在端口 8000 上以 3 个副本运行 `transformers-web` 应用程序，并为数据库凭据指定环境变量。要成功部署此应用程序，我为其他组件如网络和数据库存储创建了清单。部署后，将应用程序暴露给网络的其余部分是必要的。

### 网络和持久存储

我设置了一个 ingress 控制器和应用程序的 Postgress 数据库的存储卷。在 Kubernetes 的上下文中，入口控制器用于将 HTTP 流量路由到集群中的不同服务。这些控制器基于 Nginx、Traefik 或 HAProxy，并允许你指定例如流量到 `service-a.example.com` 应该被定向到服务 A，而流量到 `service-b.example.com` 应该被定向到服务 B.

在下面的配置中，我为 Transformers Web 应用程序在端口 8000 上设置了 HTTP 路由：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: transformers-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /$1
spec:
  ingressClassName: nginx
  rules:
  - http:
      paths:
        - path: /?(.*)
          pathType: ImplementationSpecific
          backend:
            service:
              name: transformers-service
              port:
                number: 8000
```

接下来，我设置了必要的配置以在 Postgres 数据库中存储数据。Kubernetes 抽象了底层存储，因此有必要创建配置，以便只要正确的卷可用，存储就可以在不同的环境中工作：

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: postgres-pv
spec:
  capacity:
    storage: 2Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: "/data/postgres"
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
```

此配置定义了两个 Kubernetes 资源，一个和一个 `PersistentVolumeClaim`。持久卷声明是对 Kubernetes 的存储请求，而持久卷是实际的存储资源。

## 使用 MetalLB 进行外部访问

Kubernetes 没有为裸机集群提供网络负载均衡器。要在像 AWS、Azure 和 GCP 这样的云环境中配置负载均衡器，Kubernetes 会调用云提供商的 API 来配置负载均衡器和外部 IP。一旦负载均衡器被配置，它就会在负载均衡器中填充“外部 IP”信息。当在裸机集群中运行 Kubernetes 时，负载均衡器的外部 IP 在创建时会无限期地处于“挂起”状态。解决方案是使用像 [MetalLB](https://metallb.universe.tf/) 这样的工具为你的负载均衡器创建外部 IP.

在 Layer 2 模式下，这是我为像我的简单网络使用的模式，MetalLB 使用 [ARP (地址解析协议)](https://en.wikipedia.org/wiki/Address_Resolution_Protocol) 来响应对外部 IP 地址的请求。当服务通过 MetalLB 被暴露时，它会“声称”你分配给它的 IP 池中的一个 IP，并对同一本地网络上的其他设备的 ARP 请求做出响应，从而使服务可以被外部访问。我发现它与公共 IP 和 [RFC1819 (私有) IP 地址](https://www.techtarget.com/whatis/definition/RFC-1918) 都可以工作，这很酷。对于更复杂的设置，MetalLB 可以被配置为 [BGP](https://en.wikipedia.org/wiki/Border_Gateway_Protocol) 说话者，以向网络的路由器宣布分配的 IP 地址。我对 BGP 不太了解，所以我就不多说了。

下面的图片展示了 MetalLB 的工作方式。当请求 LoadBalancer 服务时，MetalLB 会从配置的范围中分配一个 IP 地址，并使网络知道该 IP“位于”集群中：

<figure><img src="../../.gitbook/assets/image (43).png" alt=""><figcaption></figcaption></figure>

### 安装 MetalLB

我使用 Helm 将 MetalLB 添加到我的集群中，运行以下命令：

```bash
helm repo add metallb https://metallb.github.io/metallb
helm install metallb metallb/metallb
```

第一条命令告诉 Helm 在哪里可以找到 MetalLB Helm chart 仓库，第二条命令使用 Helm chart 将 MetalLB 安装到 Kubernetes 集群中。

#### **配置**

MetalLB 需要一个它可以为集群使用的 IP 地址池。我预留了 5 个 IP 地址，范围在 `10.0.0.5` 到 `10.0.0.10` 之间，这些地址不属于我的 [dnsmasq](https://thekelleys.org.uk/dnsmasq/doc.html) DHCP 池。将 IP 分配在 DHCP 范围之外可以防止与网络中动态分配的 IP 发生冲突。对于我的简单设置，一个 IP 地址就足够了，但我计划向集群中添加更多服务，我可能需要为每个服务分配不同的 IP。我创建了以下自定义资源来配置 MetalLB，并将文件命名为 `metallb-config.yaml`：

```yaml
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
    name: pool
    namespace: default # or any other namespace
spec:
    addresses:
    - 10.0.0.5-10.0.0.10

---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
    name: l2-advertisement
    namespace: default
spec:
    addresspools:
    - pool
```

这会为 MetalLB 配置一个可以使用的 IP 池，并[使用第 2 层模式](https://metallb.universe.tf/configuration/#layer-2-configuration)通告这些 IP 地址。

有关 MetalLB 以及如何配置的更多信息，请参阅[MetalLB 官方文档](https://metallb.universe.tf/)。
