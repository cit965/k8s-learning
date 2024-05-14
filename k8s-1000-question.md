# k8s 源码面试1000题

- 01 generation 和 resouceversion的区别和用途？
- 02 k8s 有哪几种 qos策略，你能详细描述其中的一种吗？底层是linux中什么原理？
- 03 删除一个namespace经历了哪些流程，越详细越好
- 04 客户端 每次 watch资源都会通过apiserver 与etcd建立一个watch链接吗？如果不是，请说说问题和解决方案。
- 05 kubelet evictPod 会排除哪种类型的？
- 06 你能说说kubectl apply 一个文件发生的完整过程吗？越详细越好
- 07 kubernetes中pod spec设置的limits request会转换为哪些参数进行限制
- 08 metrics-server的采集周期，采集链路是什么样的
- 09 kubernetes proxy中 发现长时间运行的tcp连接 如何处理invalid包 如果更优雅的解决需要修改哪个参数
