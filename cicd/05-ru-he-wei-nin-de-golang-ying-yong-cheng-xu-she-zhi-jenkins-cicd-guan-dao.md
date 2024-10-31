# 05、如何为您的 Golang 应用程序设置 Jenkins CI/CD 管道

![Jenkins CI/CD pipeline](https://mattermost.com/wp-content/uploads/2022/04/04\_Jenkins\_CI\_CD@2x.webp)



\=在开始之前，本指南中的所有示例均基于[此 GitHub 存储库](https://github.com/shadowshot-x/micro-product-go/tree/Testing-CICD)中找到的代码。

现在，让我们开始吧！

### 为 Docker 和 Golang 设置 Jenkins

拥有弹性的 Golang CI/CD 管道有助于确保您可以最大限度地缩短平均检测时间 (MTTD) 指标，因为您不会将失败的构建发送给实际客户（即，因为他们将保留在最后一个稳定构建上） 。同时，您还将减少平均解决时间 (MTTR)，这确保您能够尽快解决问题，并且通过持续交付和持续部署，更快地发布新的稳定版本。

由于自动化，所有这些都需要最少的工程师交互。换句话说，您不需要单独的团队来测试、构建和部署您的应用程序。 CI/CD 的这些实践使交付过程变得极其高效，节省了无数的工程时间（如您所知，这是相当昂贵的）。

Jenkins 是我们可以用来设置 CI/CD 的工具。它是用 Java 编写的，并且完全开源。首先，您需要在系统上安装 Jenkins。前往[Jenkins 官方页面](https://www.jenkins.io/doc/book/installing/)执行此操作。

尽管可以使用简单的 Bash 文件，但许多开源解决方案使用 Makefile 来自动执行基本任务。在最简单的层面上，Makefile 定义了一个名称下的命令列表，可以使用`make <name>`从终端运行这些命令。

CI/CD 有四个主要步骤：

1. &#x20;单元测试，
2. &#x20;功能测试，
3. 构建 Docker 镜像，以及
4. 将 Docker 镜像推送到 Dockerhub。

Makefile 如下所示：

```go
run:
   go run main.go
unit-tests:
   go test ./...
functional-tests:
   go test ./functional_tests/transformer_test.go
build:
   docker build . -t shadowshotx/product-go-micro
```

你需要安装 golang，docker  插件

![installing plugins/upgrades](https://mattermost.com/wp-content/uploads/2022/04/image2-1-1024x446.webp)

这让 Jenkins 节点知道 Docker 和 Golang 在插件中可用。接下来，您需要通过转到全局配置这些插件 `Manage Jenkins -> Global Tool Configuration.`

设置配置如下图所示。这将使 Docker 和 Golang 可用于运行 CI/CD 工作流程的 Jenkins 节点。

![jenkins global tool configuration](https://mattermost.com/wp-content/uploads/2022/04/image5-1-1024x628.webp)

现在，您可以开始构建 Jenkins CI/CD 管道。转到`Dashboard`并选择`New Item` 。在 Jenkins 上构建整个应用程序时，您可以选择 freestyle 项目；多分支管道用于具有多个分支的存储库。由于我们不需要多个分支，因此我们将为 CI/CD 构建一个管道。

![Set up a Jenkins CI/CD Pipeline](https://mattermost.com/wp-content/uploads/2022/04/image4-1-1024x664.webp)

选择 Pipeline 后，您必须配置并添加 Jenkins 将跟踪的 GitHub 存储库。这可以定期、手动或在 GitHub 事件（例如拉取请求和提交）之后完成。此时，您需要添加具有要跟踪的相应分支的 GitHub 存储库。

最后，添加我们的代码和 Jenkins 用于 GitHub 和 Dockerhub 的凭据。创建这两个凭据，如下所示：

![add global credentials for Jenkins](https://mattermost.com/wp-content/uploads/2022/04/image7-1024x601.webp)

恭喜您设置了 Jenkins 服务器！这将自动跟踪您的应用程序的 GitHub 更改。

### &#x20;定义您的 CI/CD 步骤

Jenkins 包含一个称为 Jenkinsfile 的描述性文件，它概述了要运行的命令。当您指定 GitHub 分支时，Jenkins 将自动在那里搜索 Jenkinsfile。当找到这样的文件时，Jenkins将根据该文件运行命令。

首先，我们为 Jenkins 节点设置 Golang 版本。这与我们在全局列中配置的相同。接下来，我们分阶段具体说明步骤：

* 我们运行命令`make unit-test`单元测试用例，它将引用 Makefile。
* 功能测试，之后我们将构建 Docker 镜像。
* 通过使用 Dockerhub 凭据登录并推送映像来交付映像。

```go
pipeline {
    // install golang 1.14 on Jenkins node
    agent any
    tools {
        go 'go1.14'
    }
    environment {
        GO114MODULE = 'on'
        CGO_ENABLED = 0 
        GOPATH = "${JENKINS_HOME}/jobs/${JOB_NAME}/builds/${BUILD_ID}"
    }
    stages {
        stage("unit-test") {
            steps {
                echo 'UNIT TEST EXECUTION STARTED'
                sh 'make unit-tests'
            }
        }
        stage("functional-test") {
            steps {
                echo 'FUNCTIONAL TEST EXECUTION STARTED'
                sh 'make functional-tests'
            }
        }
        stage("build") {
            steps {
                echo 'BUILD EXECUTION STARTED'
                sh 'go version'
                sh 'go get ./...'
                sh 'docker build . -t shadowshotx/product-go-micro'
            }
        }
        stage('deliver') {
            agent any
            steps {
                withCredentials([usernamePassword(credentialsId: 'dockerhub', passwordVariable: 'dockerhubPassword', usernameVariable: 'dockerhubUser')]) {
                sh "docker login -u ${env.dockerhubUser} -p ${env.dockerhubPassword}"
                sh 'docker push shadowshotx/product-go-micro'
                }
            }
        }
    }
}
```

![Jenkins CI/CD workflow](https://mattermost.com/wp-content/uploads/2022/04/image6-1024x467.webp)

![Jenkins CI/CD workflow final image](https://mattermost.com/wp-content/uploads/2022/04/image8-1024x668.webp)

### 持续交付与持续部署

恭喜您构建了 CI/CD 管道！现在，是时候继续学习，确保您了解交付和部署之间的区别。

在生产环境中手动部署镜像是业界的常见做法。当这种情况直接发生时，该过程称为持续部署。

在 Kubernetes 中可以看到 CD 的一个非常简单的场景是在设置 Pod 时始终使用镜像拉取策略。 Kubernetes 跟踪它们正在使用的 Docker 镜像的标签。因此，一旦使用跟踪标签更新镜像，Kubernetes 就会拉取镜像并更新 Pod。就像在 YAML 中设置一样简单：

```yaml
kind: Pod
apiVersion: v1

metadata:
  name: product-micro-go
  labels:
    app: product-micro-go
spec:
  containers:
  - name: product-micro-go
    image: shadowshotx/product-micro-go
    imagePullPolicy: Always
```

您在本地计算机上设置了该项目。不幸的是，在云中事情的运作方式有点不同。当存在多个节点时，请将您的计算机视为云中的一个简单节点。

最近，公司开始通过 CI/CD 推出 Kubernetes。您在本地计算机中设置的 Jenkins 实例必须是云中处理拉取请求的实际 Jenkins 服务器。对于一个非常大的生态系统，Jenkins 可以在很大程度上进行扩展。它能够并行运行一些作业，并且可以轻松配置。在设计高可用系统时，必须确保 CI/CD 也可用，因为这将成为开发工作流程中的障碍。

使用 CI/CD 的另一个好地方是为多个操作系统构建映像时。由于芯片架构不同，在 ARM 处理器（例如 Mac）上生成的二进制文件无法在 X86 处理器（例如 HP 机器）上运行。因此，必须为这两者单独创建 Docker 镜像。

幸运的是，Jenkins 提供了一个界面，您可以在其中在特定操作系统和环境上构建镜像。因此，在持续集成阶段，可以构建并最终交付多个Docker镜像。这是 CI/CD 的另一个力量。
