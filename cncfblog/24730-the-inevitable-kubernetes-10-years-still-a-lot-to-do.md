# 不可避免的 Kubernetes——10 年，仍有很多工作要做

Kubernetes 于 2014 年 6 月推出，自那时起，它在普及云原生应用程序设计和支持更多微服务部署方面发挥了巨大作用。容器部署的增长是巨大的，而 Kubernetes 对于公司管理这些部署至关重要——根据[CNCF 上一份报告](https://www.cncf.io/reports/cncf-annual-survey-2023/)的调查结果，84% 的组织正在使用或评估 Kubernetes，而 66% 的潜在和实际消费者正在使用 Kubernetes生产。

如今，Kubernetes 不仅仅是一个容器编排器。它是一个搭建平台的平台。统一的 API 使 Kubernetes 成为跨多个云和混合环境（本地云和公共云都在发挥作用）运行工作负载的绝佳工具，从而使企业能够避免供应商锁定。这反过来又提供了架构决策的灵活性，并显着降低了基础设施成本（特别是云账单）。

这种逐年增长的采用率在很短的时间内在每个组织中创造了 Kubernetes 不可避免的形象。

问题是——Kubernetes 的下一步是什么？所有问题都解决了吗？

##  **数据库复杂性**

在数据库领域，各种社区团体开始与 Kubernetes 合作，评估它如何与他们的项目配合使用，以及如何实施在 Kubernetes 上运行的项目。这些社区想要回答的关键问题是围绕在 Kubernetes 上运行数据库时存在的挑战，以及分享初始部署的最佳实践和可以使用的性能优化步骤。

起步很坎坷。通过 StatefulSets 和持久卷的第一个版本，工程师能够在 Kubernetes 中运行有状态工作负载。但是启动数据库并保持其正常运行是两个不同的挑战。在 k8s 中运行数据库的复杂性还没有被很好地理解（请参见下图，其中显示了为单个 Percona XtraDB 集群运行所需的一些 k8s 原语）。这导致了错误的假设，即 Kubernetes 仅适用于无状态工作负载。

![percona XtraDB Cluster primatives in Kubernetes](https://lh7-us.googleusercontent.com/docsz/AD_4nXcCBBsqGam4xRBa94wmEUHr2nOA2gA_Z3Ks3kIjXwEw_vQb6zSc8GupOFRbAyBD0OBkfNzTVTV9m2AFV-fr6KXyPcr10aW4_H0gHtFns6jEViix0KlC3xcGGXe10UasFVC2VS7q04UIF4O4_YI-sKaxRVc?key=NL7GJlw4XstKgRZIGAb5Bw)

但令人欣慰的是，工程师们的好奇心并没有停止。容器存储接口已经成熟，为管理员提供了更好的存储控制能力。随着 Kubernetes Operators 的推出，开发人员能够大大简化复杂应用程序（例如数据库）的部署和管理。

通过使在容器中运行 PostgreSQL、MySQL 和 MongoDB 等数据库作为云原生数据部署变得更容易，更多开发人员能够普遍采用云原生应用程序方法。这种迁移为公司带来了更多的价值和更多的机会——根据[Kubernetes 社区的数据](https://dok.community/newsroom/report-shows-increased-revenue-and-productivity-for-organizations-running-data-on-kubernetes-dok/)，83% 的受访公司将超过 10% 的收入归因于在 Kubernetes 上运行数据，三分之一的组织看到他们的生产力提高了一倍。这些公司现在在 Kubernetes 上运行更复杂的工作负载，包括分析（67%）和 AI/ML（50%）。

##  **运营商问题**

操作员确实简化了数据库部署，但更重要的是他们在第二天的操作中所做的事情——消除了执行例行任务的需要，并最大限度地减少了人为错误的可能性。

但并非所有问题都得到解决。

###  **复杂**

Operator 抽象了 Kubernetes 原语并消除了配置数据库的需要。但这并不能完全消除复杂性，仍然需要工程师连接到 Kubernetes 并与 kubectl 交互来解决问题或执行各种操作任务。

###  **多云**

如上所述，Kubernetes 对基础设施进行了抽象，并提供了统一的 API。这样，它就成为构建多云和混合云平台的理想工具。但与此同时，多云故事并不完整。有人尝试通过 Federation（著名的[已退役的 KubeFed](https://github.com/kubernetes-retired/kubefed) ）以及正在进行的项目（例如 Elotl Nova 或 Karmada）来解决多集群部署问题。

缺乏统一的解决方案迫使工程师创建自己的方法来提供多集群功能。例如，所有 Percona Operator 都允许用户为数据库设置跨集群复制，但故障检测和故障转移是手动的。

###  **多数据库**

据[Redgate](https://www.datanami.com/2024/01/23/multi-database-shops-now-the-norm-redgate-says)称，79% 的公司在其堆栈中使用两种或多种数据库技术。将其映射到 Kubernetes，需要为每个数据库技术运行一个操作员。每个 Operator 都有自己的配置模式和学习曲线。这再次增加了复杂性和操作负担。

## **未来：超越运营商**

展望未来，我们期望看到另一个抽象级别，这将帮助用户应对上面列出的问题的复杂性。我们怀疑更多的开源解决方案将以 Web 应用程序或新 API 的形式出现。这就是开源软件的力量——构建像 Kubernetes 这样的解决方案，然后使其变得更好。

Percona 是一家试图正面解决复杂性、多云和多数据库问题的公司的例子。我们最近发布了[Percona Everest](https://docs.percona.com/everest/index.html) ，这是一个开源云原生数据库平台，为用户提供在 Kubernetes 上运行和管理数据库的即点即用体验。 Percona Everest 的独特之处和值得尝试的工具是，我们正在一个平台上解决上述问题。

Percona Everest 通过提供用户友好的图形界面和强大的 API，简化了 Kubernetes 和 Operator 本身固有的复杂性。这直接解决了故障排除和 Kubernetes 管理的挑战。我们通过简化跨不同 Kubernetes 集群的多个 Operator 的编排来消除多云数据库部署的复杂性。这将多区域和跨站点复制从 YAML 难题转变为简单的单击和配置体验。 （此功能尚未推出。）

这是一些初始数据——下面是很酷的图表。

![percona everest dashboard](https://lh7-us.googleusercontent.com/docsz/AD_4nXcQ3ONWCuL6oSAeEfwyDWZwEkXFKPT1tG4JutRFSkGKfUA7mbSNLlYU48RYmW1Z1O2pQG2YTVtLiyA_CjT5mUIv-WWAM5Utsk8l2Up7B0VwIoKli5ile-9eHyk12W8t2BH93gpEWHTvSaznCnYEdZENMQIw?key=NL7GJlw4XstKgRZIGAb5Bw)

总之，随着我们看到越来越多的公司采用 Kubernetes，我们将继续寻找改进工作方式的方法，从而节省开发人员的时间和公司资金。 Percona Everest 就是一个例子，我们期待在正式发布时简化和加强其功能。为 Kubernetes 的 10 年和未来 10 年的云持续创新干杯！

当充满热情的社区聚集在一起时，开源就会蓬勃发展。你的声音至关重要！通过我们的[快速入门指南](https://docs.percona.com/everest/quickstart-guide/quick-install.html)亲身体验 Percona Everest 的强大功能。在我们的[论坛](https://forums.percona.com/c/percona-everest/81)中分享您的反馈，与其他工程师联系，并通过为我们的[GitHub 存储库](https://github.com/percona/everest)做出贡献来帮助塑造云原生数据库管理的未来。让我们一起建造 Percona Everest。