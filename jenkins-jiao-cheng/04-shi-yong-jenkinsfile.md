# 04、使用 jenkinsfile

## 创建 Jenkinsfile <a href="#chuang-jian-jenkinsfile" id="chuang-jian-jenkinsfile"></a>

`Jenkinsfile` 是一个文本文件，下面的流水线实现了基本的三阶段持续交付流水线。

```
pipeline {
    agent any

    stages {
        stage('Build') {
            steps {
                echo 'Building..'
            }
        }
        stage('Test') {
            steps {
                echo 'Testing..'
            }
        }
        stage('Deploy') {
            steps {
                echo 'Deploying....'
            }
        }
    }
}
```

## 构建

对于许多项目来说，流水线“工作”的开始就是“构建”阶段。通常流水线的这个阶段包括源代码的组装、编译或打包。`Jenkinsfile` 文件**不能**替代现有的构建工具，如 GNU/Make、Maven、Gradle 等，而应视其为一个将项目的开发生命周期的多个阶段（构建、测试、部署等）绑定在一起的粘合层。

Jenkins 有许多插件可以用于调用几乎所有常用的构建工具，不过这个例子只是从 shell 步骤（`sh`）调用 `make`。`sh` 步骤假设系统是基于 Unix/Linux 的，对于基于 Windows 的系统可以使用 `bat` 替代。

\
`sh` 步骤调用 `make` 命令，只有命令返回的状态码为零时才会继续。任何非零的返回码都将使流水线失败。`archiveArtifacts` 捕获符合模式（\`\`\*\*/target/\*.jar\`\`）匹配的交付件并将其保存到 Jenkins master 节点以供后续获取。

```
pipeline {
    agent any

    stages {
        stage('Build') {
            steps {
                sh 'make' 
                archiveArtifacts artifacts: '**/target/*.jar', fingerprint: true 
            }
        }
    }
}
```

## 测试

运行自动化测试是任何成功的持续交付过程的重要组成部分。因此，Jenkins 有许多测试记录，报告和可视化工具，这些都是由[各种插件](https://plugins.jenkins.io/?labels=report)提供的。最基本的，当测试失败时，让 Jenkins 记录这些失败以供汇报以及在 web UI 中可视化是很有用的。下面的例子使用由 [JUnit 插件](https://plugins.jenkins.io/junit)提供的 `junit` 步骤。

在下面的例子中，如果测试失败，流水线就会被标记为“不稳定”，这通过 web UI 中的黄色球表示。基于测试报告的记录，Jenkins 还可以提供历史趋势分析和可视化。

```
pipeline {
    agent any

    stages {
        stage('Test') {
            steps {
                /* `make check` 在测试失败后返回非零的退出码；
                * 使用 `true` 允许流水线继续进行
                */
                sh 'make check || true' 
                junit '**/target/*.xml' 
            }
        }
    }
}
```

## 部署

部署可以隐含许多步骤，这取决于项目或组织的要求，并且可能是从发布构建的交付件到 Artifactory 服务器，到将代码推送到生产系统的任何东西。 在示例流水线的这个阶段，“Build（构建）” 和 “Test（测试）” 阶段都已成功执行。从本质上讲，“Deploy（部署）” 阶段只有在之前的阶段都成功完成后才会进行，否则流水线会提前退出。

```
pipeline {
    agent any

    stages {
        stage('Deploy') {
            when {
              expression {
                currentBuild.result == null || currentBuild.result == 'SUCCESS' 
              }
            }
            steps {
                sh 'make publish'
            }
        }
    }
}
```
