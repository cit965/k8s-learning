# 01-通过 docker desktop 运行容器

## 容器是什么

**容器（Container）**&#x662F;一种轻量级的**虚拟化技术**，它允许在单个操作系统实例上运行多个隔离的应用程序。

**容器**技术使得应用程序及其依赖项可以作为**一个单元（容器）**&#x88AB;封装起来，这样就可以在任何支持容器技术的平台上一致地运行，而不受底层基础设施的影响。

当前容器产品在企业中的使用已经非常普遍，对于运维和开发团队而言，熟练掌握容器技术的使用也成为一门必不可少的技能。

<figure><img src="../../.gitbook/assets/image (1) (1) (1) (1) (1) (1) (1) (1) (1).png" alt="" width="563"><figcaption></figcaption></figure>

## Docker desktop

Docker Desktop 是一个为开发人员提供的桌面应用程序，它简化了在本地环境中使用 Docker 容器的过程。

在日常开发中，我们需要在本地去启动一些服务，如：redis、MySQL等，就需要去下载这些在本地去启动，操作较为繁琐。此时，我们可以使用Docker Desktop，来搭建我们需要的服务，直接在容器中去启动即可。 下图是 docker desktop 安装后的界面：

<figure><img src="../../.gitbook/assets/image (5) (1).png" alt="" width="563"><figcaption></figcaption></figure>

## 运行容器

接下来，你将学习如何运行一个容器。你将使用 Dockerfile 和一个示例应用程序创建一个镜像。

首先克隆代码：

```
git clone https://github.com/docker/welcome-to-docker
```

进入新的目录：

```
cd welcome-to-docker
```

在你的 IDE 中打开示例应用程序。注意它已经有一个 **Dockerfile**。对于你自己的项目，你需要自己创建这个文件。

<figure><img src="../../.gitbook/assets/image (2) (1) (1) (1) (1) (1).png" alt="" width="474"><figcaption></figcaption></figure>

你可以使用以下 **docker build** 命令通过 CLI 在你的项目文件夹中构建一个镜像。

```
docker build -t welcome-to-docker .
```

**-t** 标志为你的镜像指定一个名称（在这个例子中是 **welcome-to-docker**）。而 **.** 让 Docker 知道它可以在哪里找到 Dockerfile。构建完成后，镜像将出现在 **镜像** 标签中。选择镜像名称以查看其详细信息。选择 **运行** 以将其作为容器运行。在 **可选设置** 中记得指定一个端口号（比如 **8080**）。

<figure><img src="../../.gitbook/assets/image (1) (1) (1) (1) (1) (1) (1).png" alt="" width="563"><figcaption></figcaption></figure>

你也可以通过命令来启动一个容器：

```shell
docker run -d -p 8080:80 docker/welcome-to-docker
```

恭喜！您刚刚运行了您的第一个容器！ 🎉

您可以通过转到 Docker 桌面仪表板的**容器**视图来查看所有容器。

<figure><img src="../../.gitbook/assets/image (1) (1) (1) (1) (1) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

该容器运行一个显示简单网站的 Web 服务器。当处理更复杂的项目时，您将在不同的容器中运行不同的部分。例如，您可以为前端、后端和数据库运行不同的容器。

对于此容器，可以通过端口`8080`访问前端。要打开该网站，请选择**容器端口**列中的链接或访问 localhsot:8080 在您的浏览器中。

<figure><img src="../../.gitbook/assets/image (2) (1) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

## 探索容器

Docker Desktop 可让您探索容器的不同方面并与之交互。自己尝试一下。

1. 转到 Docker 桌面仪表板中的**容器**视图。
2. 选择您的容器。
3. 选&#x62E9;**“文件”**&#x9009;项卡以探索容器的独立文件系统。

<figure><img src="../../.gitbook/assets/image (3) (1) (1) (1) (1).png" alt="" width="563"><figcaption></figcaption></figure>

## 停止容器 <a href="#stop-your-container" id="stop-your-container"></a>

1. 转到 Docker 桌面仪表板中的**容器**视图。
2. 找到您想要停止的容器。
3. &#x5728;**“操作”**&#x5217;中选&#x62E9;**“停止”**&#x64CD;作。

<figure><img src="../../.gitbook/assets/image (4) (1).png" alt=""><figcaption></figcaption></figure>
