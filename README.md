### client-go examples

### pkg

pkg 目录是我封装了的一些对象和方法. 可以直接通过文件创建 deployment, pod, service, configmap 等.

### 参考

- [client-go 源码分析](https://herbguo.gitbook.io/client-go/)
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
- 



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
