# 03、pipeline 入门

流水线可以通过以下任一方式来创建：

* [通过 Blue Ocean](https://www.jenkins.io/zh/doc/book/pipeline/getting-started/#through-blue-ocean) - 在 Blue Ocean 中设置一个流水线项目后，Blue Ocean UI 会帮你编写流水线的 `Jenkinsfile` 文件并提交到源代码管理系统。
* [通过经典 UI](https://www.jenkins.io/zh/doc/book/pipeline/getting-started/#through-the-classic-ui) - 你可以通过经典 UI 在 Jenkins 中直接输入基本的流水线。
* [在代码仓库中定义](https://www.jenkins.io/zh/doc/book/pipeline/getting-started/#defining-a-pipeline-in-scm) - 你可以手动编写一个 `Jenkinsfile` 文件，然后提交到项目的源代码仓库中。

尽管 Jenkins 支持在经典 UI 中直接进入流水线，但通常认为最好的实践是在 `Jenkinsfile` 文件中定义流水线，Jenkins 之后会直接从源代码管理系统加载。

### 通过经典 UI

使用经典 UI 创建的 `Jenkinsfile` 由 Jenkins 自己保存（在 Jenkins 的主目录下）。

想要通过 Jenkins 经典 UI 创建一个基本流水线：

1. 如果有要求的话，确保你已登录进 Jenkins。
2.  从Jenkins 主页（即 Jenkins 经典 UI 的工作台），点击左上的 **新建任务**。

    ![Classic UI left column](https://www.jenkins.io/zh/doc/book/resources/pipeline/classic-ui-left-column.png)
3. 在 **输入一个任务名称**字段，填写你新建的流水线项目的名称。
4.  向下滚动并点击 **流水线**，然后点击页面底部的 **确定** 打开流水线配置页（已选中 **General** 选项）。

    ![Enter a name, click \<strong>Pipeline\</strong> and then click \<strong>OK\</strong>](https://www.jenkins.io/zh/doc/book/resources/pipeline/new-item-creation.png)
5. 点击页面顶部的 **流水线** 选项卡让页面向下滚动到 **流水线** 部分。
6. 在 **流水线** 部分, 确保 **定义** 字段显示 **Pipeline script** 选项。
7.  将你的流水线代码输入到 **脚本** 文本区域。\
    例如，复制并粘贴下面的声明式示例流水线代码（在 _Jenkinsfile ( … )_ 标题下）或者它的脚本化的版本到 **脚本** 文本区域。（下面的声明式示例将在整个过程的其余部分使用。）

    Jenkinsfile (Declarative Pipeline)

    ```groovy

    pipeline {
        agent any 
        stages {
            stage('Stage 1') {
                steps {
                    echo 'Hello world!' 
                }
            }
        }
    }
    ```

    |   | `agent` 指示 Jenkins 为整个流水线分配一个执行器（在 Jenkins 环境中的任何可用代理/节点上）和工作区。 |
    | - | --------------------------------------------------------------- |
    |   | `echo` 写一个简单的字符串到控制台输出。                                         |
    |   | `node` 与上面的 `agent` 做了同样的事情。                                    |

    ![Example Pipeline in the classic UI](https://www.jenkins.io/zh/doc/book/resources/pipeline/example-pipeline-in-classic-ui.png)
8. 点击 **保存** 打开流水线项目视图页面。
9.  在该页面, 点击左侧的 **立即构建** 运行流水线。

    ![Classic UI left column on an item](https://www.jenkins.io/zh/doc/book/resources/pipeline/classic-ui-left-column-on-item.png)
10. 在左侧的 **Build History** 下面，点击 **#1** 来访问这个特定流水线运行的详细信息。
11. 点击 **Console Output** 来查看流水线运行的全部输出。下面的输出显示你的流水线已成功运行。

### 在源码管理系统中

复杂的流水线很难在流水线配置页面 [经典 UI](https://www.jenkins.io/zh/doc/book/pipeline/getting-started/#through-the-classic-ui) 的**脚本**文本区域进行编写和维护。

为简化操作，流水线的 `Jenkinsfile` 可以在文本编辑器或集成开发环境（IDE）中进行编写并提交到源码管理系统 （可选择性地与需要 Jenkins 构建的应用程序代码放在一起）。然后 Jenkins 从源代码管理系统中检出 `Jenkinsfile` 文件作为流水线项目构建过程的一部分并接着执行你的流水线。

要使用来自源代码管理系统的 `Jenkinsfile` 文件配置流水线项目：

1. 按照 [通过经典 UI](https://www.jenkins.io/zh/doc/book/pipeline/getting-started/#through-the-classic-ui)上面的步骤定义你的流水线直到第5步（在流水线配置页面访问**流水线**部分）。
2. 从 **定义** 字段选择 **Pipeline script from SCM** 选项。
3. 从 **SCM** 字段，选择包含 `Jenkinsfile` 文件的仓库的源代码管理系统的类型。
4. 填充对应仓库的源代码管理系统的字段。\
   **Tip:** 如果你不确定给定字段应填写什么值，点击它右侧的 **?** 图标以获取更多信息。
5. 在 **脚本路径** 字段，指定你的 `Jenkinsfile` 文件的位置（和名称）。这个位置是 Jenkins 检出/克隆包括 `Jenkinsfile` 文件的仓库的位置，它应该与仓库的文件结构匹配。该字段的默认值采取名称为 "Jenkinsfile" 的 `Jenkinsfile` 文件并位于仓库的根路径。

当你更新指定的仓库时，只要流水线配置了版本管理系统的轮询触发器，就会触发一个新的构建。

### 通过 Blue Ocean

如果你刚接触 Jenkins 流水线，Blue Ocean UI 可以帮助你 [设置流水线项目](https://www.jenkins.io/zh/doc/book/blueocean/creating-pipelines)，并通过图形化流水线编辑器为你自动创建和编写流水线（即 `Jenkinsfile`）。

作为在 Blue Ocean 中设置流水线项目的一部分，Jenkins 给你项目的源代码管理仓库配置了一个安全的、经过身份验证的适当的连接。因此，你通过 Blue Ocean 的流水线编辑器在 `Jenkinsfile` 中做的任何更改都会自动的保存并提交到源代码管理系统。
