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
- 10 deployment rollback的原理是什么，你能给我说说背后发生了哪些事，涉及到的deploy spec上的哪些字段？
- 11 kubelet 中pleg 是用来解决什么问题？
- 12 pod gc 你了解吗，给我说说什么情况下pod 会被gc
- 13 pod 不允许改哪些字段？
- 14 limitrange 对于pod 的 update 事件会不会处理？
- 15 请说说 cri，containerd，runc之间的关系？执行 docker run xxx 命令底层经过哪些步骤，越详细越好。
- 16 k8s内部版本和外部版本有什么区别，etcd存的什么版本？
- 17 client 有哪几种客户端？
- 18 kubernetes shared Informer 机制
