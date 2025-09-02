---
description: 本节给大家介绍在 linux 环境下如何开发调试 k8s源码，写下这篇文章的时候，k8s 最新的版本是 1.32.0
---

# 02-借助minikube调试源码

## 1. 安装 go 语言

<figure><img src="../../.gitbook/assets/image (1) (1) (1) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

## 2. 安装 goland

<figure><img src="../../.gitbook/assets/image (59).png" alt=""><figcaption></figcaption></figure>

## 3.下载  k8s 源码

`git clone https://github.com/kubernetes/kubernetes.git`

## 4. 启动 minikube

```
minikube start --container-runtime=docker --image-mirror-country='cn'  --kubernetes-version=v1.32.0
```

## 5.调试 kube-scheduler 服务

### 1） 进入到 Minikube 容器

`docker exec -it minikube bash`

<figure><img src="../../.gitbook/assets/image (60).png" alt=""><figcaption></figcaption></figure>

### 2)    查看 kube-scheduler 启动命令

`ps -ef | grep scheduler`&#x20;

<figure><img src="../../.gitbook/assets/image (65).png" alt=""><figcaption></figcaption></figure>

kube-scheduler 启动只依赖了一个配置文件，我们将配置文件复制到宿主机源码目录下,将启动参数复制到goland编辑器中：

<figure><img src="../../.gitbook/assets/image (62).png" alt=""><figcaption></figcaption></figure>

### 3） 修改 scheduler.conf&#x20;

kube-scheduler 启动需要连接 kube-apiserver ，这里宿主机 kube-apiserver 映射的端口是 32771，所以我们需要修改 scheduler.conf 中 server 地址 8443-> 32771

<figure><img src="../../.gitbook/assets/image (63).png" alt=""><figcaption></figcaption></figure>

<figure><img src="../../.gitbook/assets/image (64).png" alt=""><figcaption></figcaption></figure>

### 4）移除 kube-scheduler

`cd /etc/kubernetes`

`manifests` 目录里存放着 K8s 所有的核心组件的 yaml 文件,因为我们要用自己本地的代码代替环境中的组件，所以环境里的组件要停止，让逻辑走到本地来。以 kube-scheduler 为例：

`mv /etc/kubernetes/manifests/kube-scheduler.yaml /etc/kubernetes/kube-scheduler.yaml`

一旦把 `kube-scheduler.yaml` 从 `manifests` 文件夹中移走，则 K8s 的 kube-schedueler pod 会删除。环境没有 kube-schedueler pod：

<figure><img src="../../.gitbook/assets/image (61).png" alt=""><figcaption></figcaption></figure>

### 5） 启动调试

<figure><img src="../../.gitbook/assets/image (67).png" alt=""><figcaption></figcaption></figure>

## 6.调试 kube-apiserver

### 1)  查看 kube-apiserver 启动参数

<figure><img src="../../.gitbook/assets/image (1) (1) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

### 2) 将配置文件拷贝到宿主机

`sudo docker cp mibikube:/var/lib/mibikube /home/z/kubernetes/cmd/kube-apiserver/minikube`

<figure><img src="../../.gitbook/assets/image (2) (1) (1).png" alt=""><figcaption></figcaption></figure>

### 3)  修改启动参数

<figure><img src="../../.gitbook/assets/image (3) (1) (1).png" alt=""><figcaption></figcaption></figure>

**4)  etcd 端口转发**

`kubectl port-forward pods/etcd-minikube 2379:2379 -n kube-system`

<figure><img src="../../.gitbook/assets/image (4).png" alt=""><figcaption></figcaption></figure>

&#x20;

### 5)  修改 kubeconfig，指定 apiserver

<figure><img src="../../.gitbook/assets/image (1) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

### 6)  启动 apiserver ，打断点

### 7)  运行 kubectl get nodes, 断点生效

<figure><img src="../../.gitbook/assets/image (2) (1).png" alt=""><figcaption></figcaption></figure>

