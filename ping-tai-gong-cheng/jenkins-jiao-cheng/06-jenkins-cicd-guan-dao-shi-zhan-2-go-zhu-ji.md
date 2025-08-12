# 06、 Jenkins CI/CD 管道实战2 【go-主机】

持续集成和持续部署 (CI/CD) 管道对于现代软件开发至关重要，可帮助团队高效地构建、测试和部署代码。本文提供了为 Go 应用程序设置 Jenkins 管道的分步指南。我们将介绍如何构建应用程序、运行单元测试并将其部署到远程服务器。

### &#x20;先决条件

1. **Jenkins 服务器**：确保 Jenkins 已安装并配置。您可以在任何服务器上进行设置，例如 vmware 或托管在数字海洋等云上。
2. **Go 应用程序**：拥有一个要构建和部署的 Go 应用程序。
3. **Git 存储库**：您的 Go 应用程序的源代码应托管在 GitHub 或其他版本控制系统上。
4. **SSH 凭据**：用于访问部署服务器的凭据。

在本文中，我在 vmware 服务器上使用了部署服务器，其 IP 地址和用户名如下：

```
nn@172.16.137.133
```

### &#x20;Go应用程序代码

下面是一个简单 HTTP 服务器的 Go 代码及其相应的单元测试：

### 主应用程序代码 ( `main.go` )

```go
package main
import (
 "fmt"
 "net/http"
)
func handler(w http.ResponseWriter, r *http.Request) {
 fmt.Fprintf(w, "Hello, World yyy v2!")
}
func main() {
 port := ":8070"
 // Print a message indicating the server is starting
 fmt.Printf("Starting server on port%s\n", port)
 http.HandleFunc("/", handler)
 http.ListenAndServe(port, nil)
 fmt.Printf("Application stopped\n")
}
```

* **描述**：此 Go 应用程序在端口 8070 上启动 HTTP 服务器并响应“Hello, World yyy v2!”任何传入的请求。

### 单元测试代码（ `main_test.go` ）

```go
package main
import (
 "net/http"
 "net/http/httptest"
 "testing"
)
func TestHandler(t *testing.T) {
 // Create a request to pass to our handler.
 req := httptest.NewRequest("GET", "http://172.16.137.133:8070", nil) //replace this with your real api url
 // Create a response recorder to record the response from the handler.
 rr := httptest.NewRecorder()
 // Call the handler with the request and recorder.
 handler(rr, req)
 // Check the status code is what we expect.
 if status := rr.Code; status != http.StatusOK {
  t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
 }
 // Check the response body is what we expect.
 expected := "Hello, World yyy v2!"
 if rr.Body.String() != expected {
  t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
 }
}
```

* **描述**：此测试检查 HTTP 处理程序函数是否返回 200 OK 状态和正确的响应正文。

### &#x20;配置 Jenkins 凭据以使用 SCP 和 SSH 部署到 Ubuntu 主机

#### 先决条件 <a href="#id-383e" id="id-383e"></a>

1. **Ubuntu 主机：**&#x6709;一个可用于部署的 Ubuntu 主机。在此示例中，我们将使用 IP 地址`172.16.137.133`和用户名`nn` 。
2. **SSH 密钥对：**&#x751F;成 SSH 密钥对以安全访问 Ubuntu 主机。

#### (1) 生成SSH密钥对 <a href="#id-0b18" id="id-0b18"></a>

```
// 打开终端：
ssh-keygen -t rsa -b 4096 -C "your_email@example.com"
cat ~/.ssh/id_rsa.pub

// 登录到远程目标部署 Ubuntu 主机并将公钥添加到~/.ssh/authorized_keys文件中：
echo "your-public-key-content" >> ~/.ssh/authorized_keys
```

#### (2) 安装 jenkins 插件

导航到`Manage Jenkins` > `Manage Plugins`

<figure><img src="../../.gitbook/assets/image (31).png" alt=""><figcaption></figcaption></figure>

转到`Available`选项卡并搜索 以下插件

<figure><img src="../../.gitbook/assets/image (29).png" alt=""><figcaption></figcaption></figure>

（3）配置凭证

**转到`Manage Jenkins` > `Manage Credentials`**&#x20;

<figure><img src="../../.gitbook/assets/image (32).png" alt=""><figcaption></figcaption></figure>

**添加新凭证**

<figure><img src="../../.gitbook/assets/image (33).png" alt=""><figcaption></figcaption></figure>

### jenkinsFile

以下是用于构建、测试和部署 Go 应用程序的`Jenkinsfile` ：

```
pipeline {
    agent any
    environment {
        SSH_CREDENTIALS_ID = 'bc013f38-40d9-4731-8ed1-23c56055cc0f' // Replace with your SSH credential ID from you jenkin dashboard
    }
    stages {
        stage('Checkout') {
            steps {
                // Pull the latest code from the Git repository
                git branch: 'master', url: 'https://github.com/novrian6/api_go_jenkins_demo.git'
            }
        }
        stage('Build') {
            steps {
                // Build the Go application
                sh 'go build -o hello-world-api'
            }
        }
        stage('Test') {
            steps {
                // Run Go unit tests
                sh 'go test -v ./...'
            }
        }
        stage('Deploy') {
            steps {
                sshagent(credentials: [SSH_CREDENTIALS_ID]) {
                    script {
                        // Commands to be executed on the remote server
                        def remoteCommands = '''
                        # Stop existing Go binary process if running
                        pkill hello-world-api || true
                        # Create target directory
                        mkdir -p ~/go_apps/production
                        # Remove old binary if exists
                        rm -f ~/go_apps/production/hello-world-api
                        # Copy the new binary
                        scp -o StrictHostKeyChecking=no -i ${SSH_AUTH_SOCK} hello-world-api nn@172.16.137.133:~/go_apps/production/
                        # Change directory and make the binary executable
                        chmod +x ~/go_apps/production/hello-world-api
                        cd ~/go_apps/production
                        # Run the new binary in the background
                        nohup ./hello-world-api > /dev/null 2>&1 &
                        '''
                        // Execute commands on the remote server
                        sh "ssh -o StrictHostKeyChecking=no nn@172.16.137.133 '${remoteCommands}'"
                    }
                }
            }
        }
    }
    post {
        always {
            cleanWs()
        }
    }
}
```

### Jenkinsfile 的解释

### &#x20;1. checkout 阶段

```
stage('Checkout') {
    steps {
        // Pull the latest code from the Git repository
        git branch: 'master', url: 'https://github.com/novrian6/api_go_jenkins_demo.git'
    }
}
```

* **用途**：从指定的 Git 存储库和分支中获取最新代码。
* **操作**：使用 Jenkins 的 Git 插件克隆存储库。

### &#x20;2. 构建阶段

```
stage('Build') {
    steps {
        // Build the Go application
        sh 'go build -o hello-world-api'
    }
}
```

* **目的**：将 Go 应用程序编译为可执行二进制文件。
* **操作**：运行`go build`命令来生成`hello-world-api`二进制文件。

### &#x20;3. 测试阶段

```
stage('Test') {
    steps {
        // Run Go unit tests
        sh 'go test -v ./...'
    }
}
```

* **目的**：确保 Go 应用程序通过所有单元测试。
* **操作**：执行`go test -v ./...`运行测试并显示详细输出。

### &#x20;4. 部署阶段

```
stage('Deploy') {
    steps {
        sshagent(credentials: [SSH_CREDENTIALS_ID]) {
            script {
                def remoteCommands = '''
                # Stop existing Go binary process if running
                pkill hello-world-api || true  
                # Create target directory
                mkdir -p ~/go_apps/production
                # Remove old binary if exists
                rm -f ~/go_apps/production/hello-world-api
                # Copy the new binary
                scp -o StrictHostKeyChecking=no -i ${SSH_AUTH_SOCK} hello-world-api nn@172.16.137.133:~/go_apps/production/
                # Change directory and make the binary executable
                chmod +x ~/go_apps/production/hello-world-api
                cd ~/go_apps/production
                # Run the new binary in the background
                nohup ./hello-world-api > /dev/null 2>&1 &
                '''
                // Execute commands on the remote server
                sh "ssh -o StrictHostKeyChecking=no nn@172.16.137.133 '${remoteCommands}'"
            }
        }
    }
}
```

* 停止应用程序的任何正在运行的实例。
* 如果部署目录不存在，则创建它。
* 将新的二进制文件复制到服务器。
* 设置权限并在后台启动应用程序。
