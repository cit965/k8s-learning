# 01-认证

k8s 集群有两类用户：serviceaccount 和 user

* serviceaccount 是 k8s 管理的用户，它们被绑定到特定的名字空间， 或者由 API 服务器自动创建，或者通过 API 调用创建。服务账号与一组以 Secret 保存的凭据相关，这些凭据会被挂载到 Pod 中，从而允许集群内的进程访问 Kubernetes API。
* User通常是在外部管理，K8S不存储用户列表——也就是说，添加/编辑/删除用户都是在外部进行，无需与K8S API交互

<figure><img src="../../.gitbook/assets/截屏2024-06-06 20.49.42.png" alt=""><figcaption></figcaption></figure>

## 认证方式

认证的方式主要有：客户端证书、密码、普通 token、bootstrap token 和 JWT 认证 (主要用于 Service Account)。认证模块会检查请求头或者客户端证书的内容，我们可以同时配置一种或几种方式对请求进行认证。多种认证方式会被依次执行，只要一种方式通过，请求便得到合法认证。当所有方式都未通过时，会返回 401 状态码并中断请求。认证解决的问题是校验访问方是否合法并识别其身份。

### 静态令牌文件 <a href="#static-token-file" id="static-token-file"></a>

当 API 服务器的命令行设置了 `--token-auth-file=SOMEFILE` 选项时，会从文件中读取持有者令牌。 目前，令牌会长期有效，并且在不重启 API 服务器的情况下无法更改令牌列表。

令牌文件是一个 CSV 文件，包含至少 3 个列：令牌、用户名和用户的 UID。 其余列被视为可选的组名。

```
token,user,uid,"group1,group2,group3"
```

当使用持有者令牌来对某 HTTP 客户端执行身份认证时，API 服务器希望看到一个名为 `Authorization` 的 HTTP 头，其值格式为 `Bearer <token>`。 持有者令牌必须是一个可以放入 HTTP 头部值字段的字符序列，至多可使用 HTTP 的编码和引用机制。 例如：如果持有者令牌为 `31ada4fd-adec-460c-809a-9e56ceb75269`，则其出现在 HTTP 头部时如下所示：

`Authorization: Bearer 31ada4fd-adec-460c-809a-9e56ceb75269`

`源码在`pkg/kubeapiserver/options/authentication.go

```go
	if o.TokenFile != nil {
		fs.StringVar(&o.TokenFile.TokenFile, "token-auth-file", o.TokenFile.TokenFile, ""+
			"If set, the file that will be used to secure the secure port of the API server "+
			"via token authentication.")
	}
	
	// BuiltInAuthenticationOptions contains all build-in authentication options for API Server
type BuiltInAuthenticationOptions struct {
	APIAudiences    []string
	Anonymous       *AnonymousAuthenticationOptions
	BootstrapToken  *BootstrapTokenAuthenticationOptions
	ClientCert      *genericoptions.ClientCertAuthenticationOptions
	OIDC            *OIDCAuthenticationOptions
	RequestHeader   *genericoptions.RequestHeaderAuthenticationOptions
	ServiceAccounts *ServiceAccountAuthenticationOptions
	TokenFile       *TokenFileAuthenticationOptions
	WebHook         *WebHookAuthenticationOptions

	AuthenticationConfigFile string

	TokenSuccessCacheTTL time.Duration
	TokenFailureCacheTTL time.Duration
}
```

### 启动引导令牌[ ](https://kubernetes.io/zh-cn/docs/reference/access-authn-authz/authentication/#bootstrap-tokens)

### 为了支持平滑地启动引导新的集群，Kubernetes 包含了一种动态管理的持有者令牌类型， 称作 **启动引导令牌（Bootstrap Token）**。 这些令牌以 Secret 的形式保存在 `kube-system` 名字空间中，可以被动态管理和创建。 控制器管理器包含的 `TokenCleaner` 控制器能够在启动引导令牌过期时将其删除。

这些令牌的格式为 `[a-z0-9]{6}.[a-z0-9]{16}`。第一个部分是令牌的 ID； 第二个部分是令牌的 Secret。你可以用如下所示的方式来在 HTTP 头部设置令牌：

```http
Authorization: Bearer 781292.db7bc3a58fc5f07e
```

你必须在 API 服务器上设置 `--enable-bootstrap-token-auth` 标志来启用基于启动引导令牌的身份认证组件。 你必须通过控制器管理器的 `--controllers` 标志来启用 TokenCleaner 控制器； 这可以通过类似 `--controllers=*,tokencleaner` 这种设置来做到。 如果你使用 `kubeadm` 来启动引导新的集群，该工具会帮你完成这些设置。

身份认证组件的认证结果为 `system:bootstrap:<令牌 ID>`，该用户属于 `system:bootstrappers` 用户组。 这里的用户名和组设置都是有意设计成这样，其目的是阻止用户在启动引导集群之后继续使用这些令牌。 这里的用户名和组名可以用来（并且已经被 `kubeadm` 用来）构造合适的鉴权策略， 以完成启动引导新集群的工作。

请参阅[启动引导令牌](https://kubernetes.io/zh-cn/docs/reference/access-authn-authz/bootstrap-tokens/)， 以了解关于启动引导令牌身份认证组件与控制器的更深入的信息，以及如何使用 `kubeadm` 来管理这些令牌。

#### OpenID Connect（OIDC）令牌 <a href="#openid-connect-tokens" id="openid-connect-tokens"></a>

#### Webhook 令牌身份认证 <a href="#webhook-token-authentication" id="webhook-token-authentication"></a>

#### 身份认证代理[ ](https://kubernetes.io/zh-cn/docs/reference/access-authn-authz/authentication/#authenticating-proxy) <a href="#authenticating-proxy" id="authenticating-proxy"></a>

#### X509 客户证书 <a href="#x509-client-certs" id="x509-client-certs"></a>

#### 服务账号令牌 <a href="#service-account-tokens" id="service-account-tokens"></a>

\
\
