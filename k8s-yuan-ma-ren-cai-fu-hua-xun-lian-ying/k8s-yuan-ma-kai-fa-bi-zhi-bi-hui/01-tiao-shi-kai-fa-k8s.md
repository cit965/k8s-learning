# 01-调试开发k8s

## 使用 Docker 构建 Kubernete

官方 release 是使用 Docker 容器构建的。要使用 Docker 构建 Kubernetes，请遵循以下说明:

### Requirements

* docker

### Key scripts

以下脚本位于 `build/` 目录中。请注意，所有脚本都必须从 Kubernetes 根目录运行

* [`build/run.sh`](https://github.com/kubernetes/kubernetes/blob/a3a49887ee73fa1108adac97a797dec02ccb00d4/build/run.sh)
* [`build/copy-output.sh`](https://github.com/kubernetes/kubernetes/blob/a3a49887ee73fa1108adac97a797dec02ccb00d4/build/copy-output.sh)
* [`build/make-clean.sh`](https://github.com/kubernetes/kubernetes/blob/a3a49887ee73fa1108adac97a797dec02ccb00d4/build/make-clean.sh)
* [`build/shell.sh`](https://github.com/kubernetes/kubernetes/blob/a3a49887ee73fa1108adac97a797dec02ccb00d4/build/shell.sh)

## 在本地操作系统/shell 环境上构建 Kubernetes【推荐】

虽然通过 Docker 构建可能更简单，但有时在本地电脑工作进行开发更好。下面的详细信息概述了在 Linux、Windows 和 macOS 上构建的硬件和软件要求。

学习 k8s 源码的第一步就是在本机搭建开发环境，下面我会教大家如何搭建，我的本机为 mac 电脑 m1 芯片, 参考链接如下

#### 硬件要求

* 8GB 内存
* 50GB磁盘

**macOS**

1. 安装一些软件

```bash
brew install coreutils ed findutils gawk gnu-sed gnu-tar grep make jq
```

2. 您需要在 `.bashrc` 或 shell 初始化脚本的末尾包含此块或类似的内容,添加完后记得 source 下：

```bash
GNUBINS="$(find `brew --prefix`/opt -type d -follow -name gnubin -print)"

for bindir in ${GNUBINS[@]}
do
  export PATH=$bindir:$PATH
done

export PATH
```

3. 安装 etcd

在项目根目录执行&#x20;

```bash
./hack/install-etcd.sh
export PATH="$GOPATH/src/k8s.io/kubernetes/third_party/etcd:${PATH}"
```

4. 通过脚本构建各组件

```bash
build/run.sh make KUBE_BUILD_PLATFORMS=darwin/arm64
```

执行完毕后你在项目下 \_output 目录下能够看到编译好的二进制

5. 运行以下命令启动各组件

```go
./hack/local-up-cluster.sh -O
```

6. 调试apiserver

```go
ps -ef | grep kube-apiserver
kill -9 apiserver
```

7. 打开goland，复制第六步的启动指令，开启调试,结果如下如

## 其他系统

参考 ： [https://github.com/kubernetes/community/blob/master/contributors/devel/development.md](https://github.com/kubernetes/community/blob/master/contributors/devel/development.md)



步骤图：

<figure><img src="../../.gitbook/assets/截屏2024-06-18 16.50.40.png" alt=""><figcaption></figcaption></figure>

<figure><img src="../../.gitbook/assets/截屏2024-06-18 16.55.07.png" alt=""><figcaption></figcaption></figure>

## 调度器goland参数参考

```bash
--authentication-kubeconfig=/Users/z/.kube/config
--authorization-kubeconfig=/Users/z/.kube/config
--bind-address=127.0.0.1
--kubeconfig=/Users/z/.kube/config
--leader-elect=false
```



<figure><img src="../../.gitbook/assets/截屏2024-07-10 12.42.54.png" alt=""><figcaption></figcaption></figure>
