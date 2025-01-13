# 01-运行在 docker 中的 windows

今天给大家介绍个非常有趣的项目 dockur/windows ，该项目能让你在 docker 中一键启动 windows，非常适合 linux ， mac 用户在有需要windows的场景下使用。

### 功能

* ISO 下载器
* KVM 加速
* 网页操作

docker 启动 ：

```
docker run -it --rm -p 8006:8006 --device=/dev/kvm --device=/dev/net/tun --cap-add NET_ADMIN --stop-timeout 120 dockurr/windows
```



Github项目地址：[https://github.com/dockur/windows](https://github.com/dockur/windows)



🌟 公众号直接点击 「加群」，可加入技术交流微信群。有兴趣的同学快快加入吧，群里有不少业界大神哟！\
\
\
📕 关注『CIT云原生』公众号，带你开启有趣新生活！更多好用好玩的软件资源，尽在这里。
