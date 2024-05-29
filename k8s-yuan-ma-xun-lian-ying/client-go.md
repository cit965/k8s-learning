# client-go

## 1、介绍

client-go 是用来和 k8s 集群交互的go语言客户端库，地址为：https://github.com/kubernetes/client-go client-go 的版本有两种标识方式：

* v0.x.y (For each `v1.x.y` Kubernetes release, the major version (first digit) would remain `0`)
* kubernetes-1.x.y

ps: k8s 发布v1.30.0 版本时，client-go 会同步发布两个版本 v0.30.0 (推荐使用)和 kubernetes-1.30.0

如果你想使用最新版本的客户端库，你可以运行如下命令：

`go get k8s.io/client-go@latest`

如果你想使用特定版本，推荐使用如下命令：

`go get k8s.io/client-go@v0.20.4`&#x20;

## 2、目录结构

client-go 包含以下几部分：

* The `kubernetes` package contains the clientset to access Kubernetes API.
* The `discovery` package is used to discover APIs supported by a Kubernetes API server.
* The `dynamic` package contains a dynamic client that can perform generic operations on arbitrary Kubernetes API objects.
* The `plugin/pkg/client/auth` packages contain optional authentication plugins for obtaining credentials from external sources.
* The `transport` package is used to set up auth and start a connection.
* The `tools/cache` package is useful for writing controllers.

## 3、使用

### 1.初始化

* 集群内初始化
* 集群外初始化

如果你的应用跑在集群中的pod上，推荐使用集群内初始化，否则使用集群外初始化。

```go
// 集群外初始化 client
// 需要集群的 kubeconfig 配置文件
func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
}
```

### 2. 列出集群内pod资源

```go
pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

```
