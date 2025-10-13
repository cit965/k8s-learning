# 2025 年如何为 Kubernetes 做出贡献

我经常被问到的一个问题是“我如何开始为 Kubernetes 做贡献？”。这个问题的答案永远是“视情况而定”。Kubernetes 项目非常庞大，贡献对不同的人来说可能意味着很多不同的事情。在提出这个问题时，了解自己希望从贡献中获得什么会大有帮助。

这篇文章汇集了多年来我向问过我这个问题的人给出的所有不同答案。



* 观看[新的贡献者培训](https://youtu.be/BsQB0JbMXmI?si=ON2P2LIiYa8Rw6HS)
* 参加 [SIG 会议](https://k8s.dev/calendar)
* 加入[邮件列表](https://groups.google.com/a/kubernetes.io/g/dev)
* 寻找门槛低的机会，例如[影子计划](https://groups.google.com/a/kubernetes.io/g/dev/c/ei-svAeA8Rg/m/B5rp8vmJAAAJ)和[导师团队](https://groups.google.com/a/kubernetes.io/g/dev/c/JbdZJVjmkwQ/m/4VW6bmhNAAAJ)
* &#x20; 重要存储库：
  * [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes) ：所有 Kubernetes 代码
  * [kubernetes/community](https://github.com/kubernetes/community) ：有关所有 SIG 的有用链接、指南和信息
  * [kubernetes/website](https://github.com/kubernetes/website) ： [kubernetes.io](https://kubernetes.io/) 的源代码，托管我们所有的文档
  * [kubernetes/enhancements](https://github.com/kubernetes/enhancements) ：所有功能的设计提案，又名 Kubernetes 增强提案（KEP）
* [k8s.dev](https://kuberentes.dev/) ，我们的贡献者网站，提供贡献者文档和指南
* 进入门槛较低的 SIG：
  * [SIG ContribEx](https://github.com/kubernetes/community/tree/master/sig-contributor-experience) ：您可以管理我们的社交媒体，帮助为贡献者网站撰写博客
  * [SIG Docs](https://github.com/kubernetes/community/tree/master/sig-docs) ：帮助将 Kubernetes 文档本地化为不同的语言，撰写和审阅博客文章
  * [SIG 发布 ](https://github.com/kubernetes/community/tree/master/sig-release)：发布团队中的 Shadow 负责督促贡献者确保 Kubernetes 按时发布
* [Kubernetes 开发者文档 ](https://github.com/kubernetes/community/tree/master/contributors/devel#table-of-contents)，开始贡献代码

***

#### 想要做出贡献但不知道从哪里开始？ <a href="#want-to-contribute-but-dont-know-where-to-start" id="want-to-contribute-but-dont-know-where-to-start"></a>

首先观看这个[新贡献者指导电话 ](https://youtu.be/BsQB0JbMXmI?si=ON2P2LIiYa8Rw6HS)。

这是每月第三个星期二举行的电话会议。本次演讲将带您了解 Kubernetes 社区的完整架构，并教您如何入门。

#### Kubernetes 社区的结构 <a href="#structure-of-the-kubernetes-community" id="structure-of-the-kubernetes-community"></a>

Kubernetes 项目分为不同的特别兴趣小组 (SIG)。每个 SIG 都有各自的主席和技术负责人，负责运营该 SIG。每个 SIG 负责项目的特定部分。例如，SIG Node 负责 Kubernetes 节点上发生的所有事情，例如 kubelet、容器运行时等。

我们可以用三种方式对 SIG 进行分组：

* **项目 SIG** ：涉及整个 Kubernetes 社区的项目范围工作，例如贡献者体验、文档、发布等。这也包括项目的非技术性、人性化的方面。
* **水平 SIG** ：广泛的技术重点（例如 SIG API Machinery，处理能够在所有不同组件之间构建 API 的基础设施，而不仅限于 kube-apiserver）
* **垂直 SIG** ：特定的技术重点（例如 SIG Etcd，专门处理改进和维护 etcd）

下图显示了所有不同的 SIG：

<figure><img src="../../.gitbook/assets/image.png" alt=""><figcaption></figcaption></figure>

#### 公开会议和社区日历 <a href="#open-meetings-and-the-community-calendar" id="open-meetings-and-the-community-calendar"></a>

我最喜欢 Kubernetes 社区的一点是，所有的会议和决策都是公开进行的（除了涉及行为准则、选举等敏感事项）。

所有 SIG 会议均在 Zoom 上公开举办，任何人都可以加入。我最喜欢与大家分享的资源之一是我们的社区日历 [k8s.dev/calendar](http://k8s.dev/calendar) 。您可以在这里找到所有 SIG 会议的邀请。所有这些会议都会录制并发布到 [Kubernetes YouTube 频道 ](https://www.youtube.com/@KubernetesCommunity)。此外，我们还提供亚太地区版本的贡献者招募活动，面向全球各地的贡献者。

#### &#x20; 影子计划 <a href="#shadow-programs" id="shadow-programs"></a>

Kubernetes 项目一直需要像您这样的新贡献者！为了方便新贡献者的加入，我们推出了多个影子项目。这些项目旨在寻找和培训新贡献者，因此加入的先决条件通常不多。以下是一些热门项目：

* [SIG Docs PR 管理员影子计划 ](https://github.com/kubernetes/website/wiki/PR-Wranglers)：PR 管理员影子计划会帮助当周的 PR 管理员审阅 [kubernetes/website](https://github.com/kubernetes/website) 的文档 PR。你可以在 Slack 的 [#sig-docs](https://kubernetes.slack.com/archives/C1J0BPD2M) 频道联系当周的 PR 管理员，询问是否可以跟随他们。这是开始贡献的最简单方法之一。我的贡献之旅也是从 PR 管理员影子开始的 :)
* [发布团队影子计划 ](https://github.com/kubernetes/sig-release/tree/master/release-team)：发布团队负责召集 Kubernetes 贡献者，并确保 Kubernetes 按时发布。我们每年发布三个版本，因此您将有三次机会申请成为任何子团队（链接页面中提到）的影子成员。
* [生产就绪审核员影子计划：生产就绪审核员 ](https://github.com/kubernetes/community/blob/master/sig-architecture/production-readiness.md#becoming-a-prod-readiness-reviewer-or-approver)(PRR) 流程确保每个版本中的功能都已做好生产就绪的准备。您需要填写一份 PRR 问卷并作答。这些答案将由生产就绪审批员进行审核。您可以影子这些 PRR 审批员。这是一个高级职位，需要您熟悉 Kubernetes 代码库以及我们遵循的最佳实践。

#### &#x20; 进一步阅读 <a href="#further-reading" id="further-reading"></a>

* [从高层次开始使用 Kubernetes](https://gist.github.com/MadhavJivrajani/e81503b2e141da32485dfd17d2cb09fb) ： [Madhav](https://github.com/MadhavJivrajani) 的精彩要点，其中包含一些关于学习容器化、Kubernetes 设计原则、Kubernetes API 内部结构等的讲座链接
* [“我想为 Kubernetes 做贡献，我该如何开始？”](https://x.com/dims/status/1329400522890219520) [dims](https://github.com/dims) 的 Twitter 帖子包含许多有用的链接
* [Kubernetes 开发者文档](https://github.com/kubernetes/community/tree/master/contributors/devel#table-of-contents)

\
