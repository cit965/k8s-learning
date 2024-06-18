# 调试开发k8s

### 使用 Docker 构建 Kubernete

官方 release 是使用 Docker 容器构建的。要使用 Docker 构建 Kubernetes，请遵循以下说明:

### Requirements

* docker

### Key scripts

以下脚本位于 `build/` 目录中。请注意，所有脚本都必须从 Kubernetes 根目录运行

* [`build/run.sh`](https://github.com/kubernetes/kubernetes/blob/a3a49887ee73fa1108adac97a797dec02ccb00d4/build/run.sh)
* [`build/copy-output.sh`](https://github.com/kubernetes/kubernetes/blob/a3a49887ee73fa1108adac97a797dec02ccb00d4/build/copy-output.sh)
* [`build/make-clean.sh`](https://github.com/kubernetes/kubernetes/blob/a3a49887ee73fa1108adac97a797dec02ccb00d4/build/make-clean.sh)
* [`build/shell.sh`](https://github.com/kubernetes/kubernetes/blob/a3a49887ee73fa1108adac97a797dec02ccb00d4/build/shell.sh)
