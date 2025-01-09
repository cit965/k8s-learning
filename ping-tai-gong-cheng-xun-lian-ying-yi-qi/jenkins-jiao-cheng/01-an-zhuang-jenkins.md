# 01、安装 jenkins

## 介绍

jenkins 是最老牌功能最全的devops平台，适合市面上99%的场景，如果你刚接触 devops，那么从jenkins出发是个不错的选择。

无论您是开发人员、DevOps 工程师还是 IT 专业人员，掌握 Jenkins 都可以彻底改变您构建、测试和部署应用程序的方式。在本 Jenkins 教程中，我们将从基础知识开始，解释 Jenkins 是什么以及它是如何工作的。我们将指导您设置 Jenkins、创建您的第一个 Pipeline，并探索其强大的持续集成和持续交付 (CI/CD) 功能。最后，您将掌握如何自动执行重复任务、缩短开发周期和维护高质量软件的能力。

## 为什么选择 jenkins

* **掌握 CI/CD 管道：**&#x9AD8;效地自动构建、测试和部署代码。
* **自动执行重复任务：**&#x8282;省时间并减少软件开发中的人为错误。
* **广泛的工具集成：**&#x53EF;与 Git、Maven、Docker 和 Kubernetes 等众多工具配合使用。
* **促进团队协作：**&#x4FC3;进持续集成和反馈，改善团队工作流程。
* **可扩展性：**&#x652F;持具有分布式构建的大型项目。
* **职业发展：**&#x5BF9; Jenkins 专业知识的高需求带来了更好的工作机会。

## 安装

Jenkins 是使用 Java 编程语言开发的，在您需要安装 Jenkins 之前，您需要在要安装 Jenkins 的系统上安装 Java。Jenkins 支持在 linux 、windows、macos 上运行。

由于 Jenkins 官方也提供了 Docker 安装文档版，这种安装方式更加简单，所以本节教程我们使用 Docker 来安装，打开官网安装文档：[https://www.jenkins.io/doc/book/installing/docker/](https://www.jenkins.io/doc/book/installing/docker/) ，按照官方安装说明安装即可。

### 软件运行所需资源

* &#x20;256 MB 内存
* 1 GB 磁盘空间

### 安装步骤

```
docker network create jenkins
```

<pre><code>// 启动 dind 镜像，需要确保网络畅通
docker run --name jenkins-docker --rm --detach \
  --privileged --network jenkins --network-alias docker \
<strong>  --env DOCKER_TLS_CERTDIR=/certs \
</strong>  --volume jenkins-docker-certs:/certs/client \
  --volume jenkins-data:/var/jenkins_home \
  --publish 2376:2376 \
  docker:dind --storage-driver overlay2
</code></pre>

```
// 建一个 docekrfile ，将下面内容粘贴进文件

FROM jenkins/jenkins:2.479.1-jdk17
USER root
RUN apt-get update && apt-get install -y lsb-release
RUN curl -fsSLo /usr/share/keyrings/docker-archive-keyring.asc \
  https://download.docker.com/linux/debian/gpg
RUN echo "deb [arch=$(dpkg --print-architecture) \
  signed-by=/usr/share/keyrings/docker-archive-keyring.asc] \
  https://download.docker.com/linux/debian \
  $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list
RUN apt-get update && apt-get install -y docker-ce-cli
USER jenkins
RUN jenkins-plugin-cli --plugins "blueocean docker-workflow"
```

```
// 构建镜像
docker build -t myjenkins-blueocean:2.479.1-1 .
```

```
docker run --name jenkins-blueocean --restart=on-failure --detach \
  --network jenkins --env DOCKER_HOST=tcp://docker:2376 \
  --env DOCKER_CERT_PATH=/certs/client --env DOCKER_TLS_VERIFY=1 \
  --publish 8080:8080 --publish 50000:50000 \
  --volume jenkins-data:/var/jenkins_home \
  --volume jenkins-docker-certs:/certs/client:ro \
  myjenkins-blueocean:2.479.1-1
```

### 安装后设置

打开 localhost:8080, 来到解锁界面，这里需要获取管理员密码，可以执行下面命令获取到：

```sh
sudo docker exec ${CONTAINER_ID or CONTAINER_NAME} cat /var/jenkins_home/secrets/initialAdminPassword 
```

<figure><img src="../../.gitbook/assets/image (28).png" alt=""><figcaption></figcaption></figure>

[解锁 Jenkins](https://www.jenkins.io/doc/book/installing/docker/#unlocking-jenkins)后，会出&#x73B0;**“自定义 Jenkins”**&#x9875;面。作为初始设置的一部分，您可以在此处安装任意数量的有用插件。这里我们选择第一个，仅仅安装默认插件。

<figure><img src="../../.gitbook/assets/1730336059284.png" alt=""><figcaption></figcaption></figure>

选择完成后，插件开始安装，耐心等待即可：

<figure><img src="../../.gitbook/assets/1730336100806.png" alt=""><figcaption></figcaption></figure>

插件安装完成后会要求我们创建一个管理员账号，输入账号密码邮箱即可：

<figure><img src="../../.gitbook/assets/1730336283030.png" alt=""><figcaption></figcaption></figure>

创建完管理员账号后，会要求我们确认回调地址，回调地址用来接收一些外部系统的调用，包括电子邮件通知、PR 状态更新以及为构建步骤提供的`BUILD_URL`环境变量。你需要看看默认的地址是否合适：

<figure><img src="../../.gitbook/assets/1730336463931.png" alt=""><figcaption></figcaption></figure>

到此 Jenkins 已经安装完毕，下面可以正式开始我们的 jenkins 学习之旅了！

<figure><img src="../../.gitbook/assets/1730336519268.png" alt=""><figcaption></figcaption></figure>
