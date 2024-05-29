# 01-client-go

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

上述代码在client-go仓库下[examples目录](https://github.com/kubernetes/client-go/tree/master/examples/out-of-cluster-client-configuration)

你可以运行该代码：

```bash
cd out-of-cluster-client-configuration
go build -o app .
./app -kubeconfig=/path/to/xxx
```

### 3. deployment资源CRUD - typed (clientset)方式

以下代码创建一个有2个副本数的deployment，然后更新副本数为1，升级nginx镜像版本，最后删除

```go
	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	fmt.Println("Creating deployment...")
	result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())

	// Update Deployment
	prompt()
	fmt.Println("Updating deployment...")
	//    You have two options to Update() this Deployment:
	//
	//    1. Modify the "deployment" variable and call: Update(deployment).
	//       This works like the "kubectl replace" command and it overwrites/loses changes
	//       made by other clients between you Create() and Update() the object.
	//    2. Modify the "result" returned by Get() and retry Update(result) until
	//       you no longer get a conflict error. This way, you can preserve changes made
	//       by other clients between Create() and Update(). This is implemented below
	//			 using the retry utility package included with client-go. (RECOMMENDED)
	//
	// More Info:
	// https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the latest version of Deployment before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		result, getErr := deploymentsClient.Get(context.TODO(), "demo-deployment", metav1.GetOptions{})
		if getErr != nil {
			panic(fmt.Errorf("Failed to get latest version of Deployment: %v", getErr))
		}

		result.Spec.Replicas = int32Ptr(1)                           // reduce replica count
		result.Spec.Template.Spec.Containers[0].Image = "nginx:1.13" // change nginx version
		_, updateErr := deploymentsClient.Update(context.TODO(), result, metav1.UpdateOptions{})
		return updateErr
	})
	if retryErr != nil {
		panic(fmt.Errorf("Update failed: %v", retryErr))
	}
	fmt.Println("Updated deployment...")

	// List Deployments
	prompt()
	fmt.Printf("Listing deployments in namespace %q:\n", apiv1.NamespaceDefault)
	list, err := deploymentsClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, d := range list.Items {
		fmt.Printf(" * %s (%d replicas)\n", d.Name, *d.Spec.Replicas)
	}

	// Delete Deployment
	prompt()
	fmt.Println("Deleting deployment...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := deploymentsClient.Delete(context.TODO(), "demo-deployment", metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
	fmt.Println("Deleted deployment.")
}

func prompt() {
	fmt.Printf("-> Press Return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Println()
}

func int32Ptr(i int32) *int32 { return &i }

```

### 4.deployment资源CRUD - dynamic方式

使用typed方式创建资源虽然方便，但是需要事先使用client-gen生成且会与某个特定版本耦合，不够通用，所以我们可以使用 unstructured.Unstructured 来表示集群中任意对象。

```go
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	
	deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": "demo-deployment",
			},
			"spec": map[string]interface{}{
				"replicas": 2,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "demo",
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "demo",
						},
					},

					"spec": map[string]interface{}{
						"containers": []map[string]interface{}{
							{
								"name":  "web",
								"image": "nginx:1.12",
								"ports": []map[string]interface{}{
									{
										"name":          "http",
										"protocol":      "TCP",
										"containerPort": 80,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	fmt.Println("Creating deployment...")
	result, err := client.Resource(deploymentRes).Namespace(apiv1.NamespaceDefault).Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created deployment %q.\n", result.GetName())

```

### 5. discoveryClient

3-4小节我们学习了 typed client 和 dynamic client ，这两个client使用来操作集群中资源，比如创建 deployment，删除pod。discovery client 不同，他是用来发现服务端支持的组、版本、资源类型：

```
// Package discovery provides ways to discover server-supported
// API groups, versions and resources.
```

我们可以查询服务端支持的组与版本、某个组的资源列表

```go
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

	// *查询服务端支持的组列表
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(config)
	gs, err := discoveryClient.ServerGroups()
	if err != nil {
		panic(err.Error())
	}
	spew.Dump(gs.Groups)
	
	// *查询core/v1 组下的资源列表
	rs, err := discoveryClient.ServerResourcesForGroupVersion("v1")
	if err != nil {
		panic(err.Error())
	}
	for _, r := range rs.APIResources {
		fmt.Println(r.Name)
	}

}

// 输出支持的组列表
/*
(v1.APIGroup) &APIGroup{Name:apps,Versions:[]GroupVersionForDiscovery{GroupVersionForDiscovery{GroupVersion:apps/v1,Version:v1,},},PreferredVersion:GroupVersionForDiscovery{GroupVersion:apps/v1,Version:v1,},ServerAddressByClientCIDRs:[]ServerAddressByClientCIDR{},},
(v1.APIGroup) &APIGroup{Name:events.k8s.io,Versions:[]GroupVersionForDiscovery{GroupVersionForDiscovery{GroupVersion:events.k8s.io/v1,Version:v1,},},PreferredVersion:GroupVersionForDiscovery{GroupVersion:events.k8s.io/v1,Version:v1,},ServerAddressByClientCIDRs:[]ServerAddressByClientCIDR{},},
...
*/

// 输出core/v1组下的资源列表
/*
pods/status
podtemplates
replicationcontrollers
replicationcontrollers/scale
replicationcontrollers/status
resourcequotas
resourcequotas/status
secrets
serviceaccounts
...
*/

```

### 6. restClient

上面三种客户端都是基于 restClient 封装的，restClient 是 client-go 最基础的客户端，对http request 进行了封装，更加灵活，一般我们用不到。

```go
// 根据配置信息构建restClient实例
	restClient, err := rest.RESTClientFor(config)

	if err!=nil {
		panic(err.Error())
	}

	// 保存pod结果的数据结构实例
	result := &corev1.PodList{}

	//  指定namespace
	namespace := "kube-system"
	// 设置请求参数，然后发起请求
	// GET请求
	err = restClient.Get().
		//  指定namespace，参考path : /api/v1/namespaces/{namespace}/pods
		Namespace(namespace).
		// 查找多个pod，参考path : /api/v1/namespaces/{namespace}/pods
		Resource("pods").
		// 指定大小限制和序列化工具
		VersionedParams(&metav1.ListOptions{Limit:100}, scheme.ParameterCodec).
		// 请求
		Do(context.TODO()).
		// 结果存入result
		Into(result)

```

## 4、局限

使用 client 操作集群中资源会导致频繁的轮询，k8s client-go 包提供了更加高效的方式：informer 。

informer 提供了一种机制来监视 Kubernetes 集群内资源的变化并做出反应。它使开发人员能够接收有关各种 Kubernetes 对象（例如 Pod、服务、部署等）状态的实时更新。

Informer 将集群中的资源缓存在本地来减少频繁的API调用，提高性能并优化资源利用率。
