### pkg

pkg 目录是我封装了的一些对象和方法. 可以直接通过文件创建 deployment, pod, service, configmap 等.



### Interafce

```golang
type HandlerInterface interface {
	CreateFromRaw(raw map[string]interface{}) (interface{}, error)
	CreateFromBytes(data []byte) (interface{}, error)
	CreateFromFile(path string) (interface{}, error)
	Create(path string) (interface{}, error)

	UpdateFromRaw(raw map[string]interface{}) (interface{}, error)
	UpdateFromBytes(data []byte) (interface{}, error)
	UpdateFromFile(path string) (interface{}, error)
	Update(path string) (interface{}, error)

	ApplyFromRaw(raw map[string]interface{}) (interface{}, error)
	ApplyFromBytes(data []byte) (interface{}, error)
	ApplyFromFile(path string) (interface{}, error)
	Apply(path string) (interface{}, error)

	DeleteByName(data []byte) error
	DeleteFromBytes(data []byte) error
	DeleteFromFile(path string) error
	Delete(name string) error

	GetByName(name string) (interface{}, error)
	GetFromBytes(name string) (interface{}, error)
	GetFromFile(path string) (interface{}, error)
	Get(name string) (interface{}, error)

	ListByLabel(label string) (interface{}, error)
	ListAll() (interface{}, error)
	List(label string) (interface{}, error)

	WatchByName(name string, addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) error
	WatchByLabel(label string, addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) error
	Watch(name string, addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) error
}
```





### 参考

- [client-go 源码分析](https://herbguo.gitbook.io/client-go/)
- [cliet-go 代码详解](https://gzh.readthedocs.io/en/latest/blogs/Kubernetes/2020-10-11-client-go系列之1---client-go代码详解.html)
- [k8sutil](https://github.com/forbearing/k8sutil)
- [client-go-parse-yaml](https://gist.github.com/pytimer/0ad436972a073bb37b8b6b8b474520fc)
- [Kubernetes client-go 系列](https://blog.csdn.net/qq_36407557/category_11210359.html)
- [如何用 client-go 拓展 Kubernetes 的 API](https://mp.weixin.qq.com/s?__biz=MzU1OTAzNzc5MQ==&mid=2247484052&idx=1&sn=cec9f4a1ee0d21c5b2c51bd147b8af59&chksm=fc1c2ea4cb6ba7b283eef5ac4a45985437c648361831bc3e6dd5f38053be1968b3389386e415&scene=21#wechat_redirect)
- [OpenShift Go Client Library Reference](https://miminar.fedorapeople.org/_preview/openshift-enterprise/registry-redeploy/go_client/getting_started.html#kubernetes-type-definitions) 
- [k8s-client-go](https://github.com/owenliang/k8s-client-go)
- [kube-client-example](https://github.com/cliterb/kube-client-example)
- [client-go 实战的文章](https://xinchen.blog.csdn.net/article/details/113753087)、 [client-go 实战的代码](https://github.com/zq2599/blog_demos/tree/master/client-go-tutorials)
- [client-go/example](https://github.com/kubernetes/client-go/tree/master/examples)
- [Kubernetes API Reference Docs](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/)
- [Working with kubernetes configmaps, part 1: volume mounts](https://itnext.io/working-with-kubernetes-configmaps-part-1-volume-mounts-f0ace283f5aa)
- [Working with kubernetes configmaps, part 2: Watchers](https://itnext.io/working-with-kubernetes-configmaps-part-2-watchers-b6dd0e583d71)
- [configmap watch example](https://github.com/ScarletTanager/configmap-watcher-example)
- [When using client-go watch api to watch deployment, result channel will auto-closed after around 5 mins #623](https://github.com/kubernetes/client-go/issues/623)
- [clien-go/tools/watch](https://github.com/kubernetes/client-go/tree/master/tools/watch)
- [kubectl and client-go](http://yuezhizizhang.github.io/kubernetes/kubectl/client-go/2020/05/13/kubectl-client-go-part-1.html)
- [client-go](https://pkg.go.dev/k8s.io/client-go)
- [can we emulate "kubectl apply" using client-go](https://github.com/kubernetes/client-go/issues/216)
- [v2 API proposal "desired vs actual" #17333](https://github.com/kubernetes/kubernetes/issues/17333)
- [Executing Remote Processes](https://miminar.fedorapeople.org/_preview/openshift-enterprise/registry-redeploy/go_client/executing_remote_processes.html)
- [如何使用client-go访问k8s crd](https://gzh.readthedocs.io/en/latest/blogs/Kubernetes/2020-09-26-使用client-go访问k8s集群中的CRD.html)
- [huweihuang client-go 笔记](https://www.huweihuang.com/kubernetes-notes/develop/client-go.html)
- [Kubernetes Informer 详解](https://www.kubernetes.org.cn/2693.html)
- [How to retrieve kubernetes metrics via client-go and golang](https://stackoverflow.com/questions/52029656/how-to-retrieve-kubernetes-metrics-via-client-go-and-golang)

### 关于 Group、Version、Resource

<img src="docs/pics/gvr-1.jpeg" alt="gvr-1" style="zoom:50%;" />

#### API Groups and their Version

```bash
# 相关的 kubectl 命令
kubectl api-resources
kubectl api-versions
```

| Group                          | Version                 |
| :----------------------------- | :---------------------- |
| `admissionregistration.k8s.io` | `v1`                    |
| `apiextensions.k8s.io`         | `v1`                    |
| `apiregistration.k8s.io`       | `v1`                    |
| `apps`                         | `v1`                    |
| `authentication.k8s.io`        | `v1`                    |
| `authorization.k8s.io`         | `v1`                    |
| `autoscaling`                  | `v1, v2beta2, v2beta1`  |
| `batch`                        | `v1, v1beta1`           |
| `certificates.k8s.io`          | `v1`                    |
| `coordination.k8s.io`          | `v1`                    |
| `core`                         | `v1`                    |
| `discovery.k8s.io`             | `v1, v1beta1`           |
| `events.k8s.io`                | `v1, v1beta1`           |
| `flowcontrol.apiserver.k8s.io` | `v1beta1`               |
| `internal.apiserver.k8s.io`    | `v1alpha1`              |
| `networking.k8s.io`            | `v1`                    |
| `node.k8s.io`                  | `v1, v1beta1, v1alpha1` |
| `policy`                       | `v1, v1beta1`           |
| `rbac.authorization.k8s.io`    | `v1, v1alpha1`          |
| `scheduling.k8s.io`            | `v1, v1alpha1`          |
| `storage.k8s.io`               | `v1, v1beta1, v1alpha1` |
