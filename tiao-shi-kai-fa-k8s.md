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

### 在本地操作系统/shell 环境上构建 Kubernetes【推荐】

虽然通过 Docker 构建可能更简单，但有时在本地电脑工作进行开发更好。下面的详细信息概述了在 Linux、Windows 和 macOS 上构建的硬件和软件要求。

#### 硬件要求

* 8GB 内存
* 50GB磁盘

**macOS**

```bash
brew install coreutils ed findutils gawk gnu-sed gnu-tar grep make jq
```

您需要在 `.bashrc` 或 shell 初始化脚本的末尾包含此块或类似的内容：

```bash
GNUBINS="$(find `brew --prefix`/opt -type d -follow -name gnubin -print)"

for bindir in ${GNUBINS[@]}
do
  export PATH=$bindir:$PATH
done

export PATH
```

#### 安装所需软件

* **GNU Development Tools GNU 开发工具**
* **Docker**
* **rsync**
*   **etcd**&#x20;

    > ```
    > export PATH="$GOPATH/src/k8s.io/kubernetes/third_party/etcd:${PATH}"
    > ```
