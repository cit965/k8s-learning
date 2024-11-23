# Table of contents

* [k8s-learning](README.md)
* [k8s 源码面试1000题](k8s-1000-question.md)

## k8s源码人才孵化训练营

* [第零章 ：阅读源码必知必会](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/k8s-yuan-ma-kai-fa-bi-zhi-bi-hui/README.md)
  * [01-调试开发k8s](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/k8s-yuan-ma-kai-fa-bi-zhi-bi-hui/01-tiao-shi-kai-fa-k8s.md)
  * [02- 当你运行 kubectl create deployment 命令时发生了什么](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/k8s-yuan-ma-kai-fa-bi-zhi-bi-hui/02-dang-ni-yun-xing-kubectl-create-deployment-ming-ling-shi-fa-sheng-le-shen-me.md)
* [第一章： client-go](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/client-go/README.md)
  * [01-client-go](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/client-go/01-client-go.md)
  * [02-Informer](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/client-go/02-informer.md)
  * [03-写一个控制器](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/client-go/03-xie-yi-ge-kong-zhi-qi.md)
  * [04-Reflector](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/client-go/04-reflector.md)
  * [05-Indexer](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/client-go/05-indexer.md)
  * [06-DeltaFIFO](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/client-go/06-deltafifo.md)
  * [07-workqueue](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/client-go/07-workqueue.md)
  * [08-sharedProcessor](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/client-go/08-sharedprocessor.md)
* [第二章 ：scheduler 源码](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/ren-zheng-shou-quan/schedule/README.md)
  * [01-初识 Scheduler](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/ren-zheng-shou-quan/schedule/01-chu-shi-scheduler.md)
  * [02-Schedule Framework](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/ren-zheng-shou-quan/schedule/02-schedule-framework.md)
  * [03-Schedule 源码讲解](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/ren-zheng-shou-quan/schedule/03schedule-yuan-ma-jiang-jie.md)
  * [04-Scheduler 二次开发](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/ren-zheng-shou-quan/schedule/04scheduler-er-ci-kai-fa.md)
  * [05-nodeSelector,nodeAffinity](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/ren-zheng-shou-quan/schedule/05-nodeselector-nodeaffinity.md)
* [第三章： operator开发](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/operator/README.md)
  * [02-operator 二次开发背景](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/operator/02operator-er-ci-kai-fa-bei-jing.md)
  * [01-controller-runtime](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/operator/01-controller-runtime.md)
* [第四章 ：controller-manager 源码](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/README.md)
  * [4.1 controller-manager 介绍](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.1-controllermanager-jie-shao.md)
  * [4.2 controller-manager 代码](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.2-controllermanager-dai-ma.md)
  * [4.3 deployment controller 01](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.3-deployment-controller-01.md)
  * [4.4 deployment controller 02](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.4-deployment-controller-02.md)
  * [4.5 deployment controller 03](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.5-deployment-controller-03.md)
  * [4.6 replicaset controller 01](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.6-replicaset-controller-01.md)
  * [4.7 replicaset controller 02](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.7-replicaset-controller-02.md)
  * [4.8 replicaset controller 03](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.8-replicaset-controller-03.md)
  * [4.9 Kubernetes Service](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.9-kubernetes-service.md)
  * [4.10 endpoint controller](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.10-endpoint.md)
  * [4.11 endpointSlice controller](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.11-endpointslice-controller.md)
  * [4.12 statefulset controller分析](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.12-statefulset-controller-fen-xi.md)
  * [4.13 daemonset controller源码分析](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.13-daemonset-controller-yuan-ma-fen-xi.md)
  * [4.16 k8s GC](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/controller-manager/4.16-k8s-gc.md)
* [第五章： apiserver 源码解析](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/README.md)
  * [5.1 核心数据结构分析](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.1-k8s-he-xin-shu-ju-jie-gou-fen-xi.md)
  * [5.2   apimachinery 初识](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.2-shen-me-shi-apimachinery.md)
  * [5.3 api-conventions（翻译）](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.2-api-conventions.md)
  * [5.6 服务端和客户端 apply](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.6-fu-wu-duan-he-ke-hu-duan-apply.md)
  * [5.7 pod 生命周期和 conditions 浅析](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.7-pod-sheng-ming-zhou-qi-he-conditions-qian-xi.md)
  * [5.8 event 源码解析](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.8-event-yuan-ma-jie-xi.md)
  * [5.10 API 组和版本控制初探](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.10-api-zu-he-ban-ben-kong-zhi-chu-tan.md)
  * [5.11 scheme 初识](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.11-scheme-chu-shi.md)
  * [5.12 k8s api-changes（翻译）](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.12-k8s-apichanges-fan-yi.md)
  * [5.13 序列化器与序列化器工厂](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.13-xu-lie-hua-qi-yu-xu-lie-hua-qi-gong-chang.md)
  * [5.14 序列化器与序列化器工厂2](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.14-xu-lie-hua-qi-yu-xu-lie-hua-qi-gong-chang-2.md)
  * [5.15 序列化器与序列化工厂3](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.15-xu-lie-hua-qi-yu-xu-lie-hua-gong-chang-3.md)
  * [5.16 apimachinery在 client-go中的使用](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.16-apimachinery-zai-clientgo-zhong-de-shi-yong.md)
  * [5.17 apimachery 其他模块](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.17-apimachery-qi-ta-mo-kuai.md)
  * [5.18 apiserver 源码分析 01](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-wu-zhang-apimachinery/5.18-apiserver-yuan-ma-fen-xi-01.md)
* [第六章： kubelet 源码分析](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-liu-zhang-kubelet-yuan-ma-fen-xi.md)
* [第七章： kubeproxy 源码分析](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-qi-zhang-kubeproxy-yuan-ma-fen-xi.md)
* [第八章： kubectl 源码分析](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-ba-zhang-kubectl-yuan-ma-fen-xi.md)
* [第九章： cloud-controller-manager 源码分析](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-jiu-zhang-cloudcontrollermanager-yuan-ma-fen-xi.md)
* [第十章： 认证授权](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/ren-zheng-shou-quan/README.md)
  * [01-认证](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/ren-zheng-shou-quan/01-ren-zheng.md)
* [第十一章：gpu](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-shi-yi-zhang-gpu/README.md)
  * [在 k8s 上安装 gpu](k8s-yuan-ma-ren-cai-fu-hua-xun-lian-ying/di-shi-yi-zhang-gpu/zai-k8s-shang-an-zhuang-gpu.md)

## 开发者要懂的 k8s

* [1 在 k8s 上部署应用](kai-fa-zhe-yao-dong-de-k8s/1-zai-k8s-shang-bu-shu-ying-yong.md)
* [Page](kai-fa-zhe-yao-dong-de-k8s/page.md)

## k8s源码每日新闻

* [2024-9-13](k8s-yuan-ma-mei-ri-xin-wen/2024-9-13.md)
* [2024-9-14](k8s-yuan-ma-mei-ri-xin-wen/2024-9-14.md)

***

* [helm 教程](helm-jiao-cheng.md)

## jenkins 教程

* [01、安装 jenkins](jenkins-jiao-cheng/01-an-zhuang-jenkins.md)
* [02、创建你的第一条pipeline](jenkins-jiao-cheng/02-chuang-jian-ni-de-di-yi-tiao-pipeline.md)
* [03、pipeline 入门](jenkins-jiao-cheng/03pipeline-ru-men.md)
* [04、使用 jenkinsfile](jenkins-jiao-cheng/04-shi-yong-jenkinsfile.md)
* [05、Jenkins CI/CD 管道实战1 【go-k8s】](jenkins-jiao-cheng/05jenkins-cicd-guan-dao-shi-zhan-1-gok8s.md)
* [06、 Jenkins CI/CD 管道实战2 【go-主机】](jenkins-jiao-cheng/06-jenkins-cicd-guan-dao-shi-zhan-2-go-zhu-ji.md)

## c# 编程入门

* [01、c# 之旅](c-bian-cheng-ru-men/01c-zhi-l.md)
* [02、VScode编辑器上编译C#](c-bian-cheng-ru-men/02vscode-bian-ji-qi-shang-bian-yi-c.md)

## python 从入门到放弃

* [python 教程](python-cong-ru-men-dao-fang-qi/python-jiao-cheng.md)
