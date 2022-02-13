### client-go examples



### 参考

- [k8s-client-go](https://github.com/owenliang/k8s-client-go)
- [kube-client-example](https://github.com/cliterb/kube-client-example)
- [client-go 实战的文章](https://xinchen.blog.csdn.net/article/details/113753087)、 [client-go 实战的代码](https://github.com/zq2599/blog_demos/tree/master/client-go-tutorials)
- [client-go/example](https://github.com/kubernetes/client-go/tree/master/examples)
- [Kubernetes API Reference Docs](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/)



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
