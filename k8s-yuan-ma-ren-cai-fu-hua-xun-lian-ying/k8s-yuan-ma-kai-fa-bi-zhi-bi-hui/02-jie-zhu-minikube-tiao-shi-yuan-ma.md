---
description: 本节给大家介绍在 linux 环境下如何开发调试 k8s源码，写下这篇文章的时候，k8s 最新的版本是 1
---

# 02-借助minikube调试源码

### 1. 安装 go 语言

<figure><img src="../../.gitbook/assets/image (1).png" alt=""><figcaption></figcaption></figure>

### 2. 安装 goland



### 3. 下载  k8s 源码

`git clone https://github.com/kubernetes/kubernetes.git`

### 1. 启动 minikube

```
minikube start --container-runtime=docker --image-mirror-country='cn'  --kubernetes-version=v1.32.0
```

其中 `--kubernetes-version` 可以选择其他版本，我这里用的是 `v1.32.0`
