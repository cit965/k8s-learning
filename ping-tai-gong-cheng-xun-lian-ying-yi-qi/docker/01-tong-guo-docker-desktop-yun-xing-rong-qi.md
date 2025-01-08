# 01-通过 docker desktop 运行容器

## 容器是什么

**容器（Container）**&#x662F;一种轻量级的**虚拟化技术**，它允许在单个操作系统实例上运行多个隔离的应用程序。

**容器**技术使得应用程序及其依赖项可以作为**一个单元（容器）**&#x88AB;封装起来，这样就可以在任何支持容器技术的平台上一致地运行，而不受底层基础设施的影响。

当前容器产品在企业中的使用已经非常普遍，对于运维和开发团队而言，熟练掌握容器技术的使用也成为一门必不可少的技能。

<figure><img src="../../.gitbook/assets/image (1).png" alt="" width="563"><figcaption></figcaption></figure>

## Docker desktop

Docker Desktop 是一个为开发人员提供的桌面应用程序，它简化了在本地环境中使用 Docker 容器的过程。

在日常开发中，我们需要在本地去启动一些服务，如：redis、MySQL等，就需要去下载这些在本地去启动，操作较为繁琐。此时，我们可以使用Docker Desktop，来搭建我们需要的服务，直接在容器中去启动即可。 下图是 docker desktop 安装后的界面：

<figure><img src="../../.gitbook/assets/image.png" alt="" width="563"><figcaption></figcaption></figure>

## 了解容器

了解容器的最佳方式是先看看它是如何工作的,你可以在 **容器** 标签下查看它（**welcome-to-docker**）。

容器一旦启动会一直运行，直到你停止它。只需点击你容器的 **停止** 图标即可。

下图是一个简单的 Web 应用程序。在 **端口** 列中选择 8088:80 来查看它正在运行。

<figure><img src="../../.gitbook/assets/image (4).png" alt="" width="563"><figcaption></figcaption></figure>

容器是一个隔离的环境，用于运行任何代码。选择该容器，然后转到 **文件** 标签查看其中的内容。

<figure><img src="../../.gitbook/assets/image (5).png" alt="" width="563"><figcaption></figcaption></figure>

你刚刚看到了一个容器在运行。接下来，你将学习如何运行一个容器。你将使用 Dockerfile 和一个示例应用程序创建一个镜像。

首先克隆代码：

```
git clone https://github.com/docker/welcome-to-docker
```

进入新的目录：

```
cd welcome-to-docker
```

在你的 IDE 中打开示例应用程序。注意它已经有一个 **Dockerfile**。对于你自己的项目，你需要自己创建这个文件。

<figure><img src="../../.gitbook/assets/image (2).png" alt="" width="474"><figcaption></figcaption></figure>

你可以使用以下 **docker build** 命令通过 CLI 在你的项目文件夹中构建一个镜像。

```
docker build -t welcome-to-docker .
```

**Breaking down this command**

**-t** 标志为你的镜像指定一个名称（在这个例子中是 **welcome-to-docker**）。而 **.** 让 Docker 知道它可以在哪里找到 Dockerfile。构建完成后，镜像将出现在 **镜像** 标签中。选择镜像名称以查看其详细信息。选择 **运行** 以将其作为容器运行。在 **可选设置** 中记得指定一个端口号（比如 **8089**）。

<figure><img src="../../.gitbook/assets/image (1) (1).png" alt="" width="514"><figcaption></figcaption></figure>

现在你有一个正在运行的容器。如果你没有为你的容器指定名称，Docker 会为你提供一个。通过点击容器名称下方的链接查看你的容器正在运行。

<figure><img src="../../.gitbook/assets/image (2) (1).png" alt="" width="313"><figcaption></figcaption></figure>

你已经学会了如何从单个镜像运行一个容器。接下来，学习如何从 Docker Hub 运行其他人的镜像。
