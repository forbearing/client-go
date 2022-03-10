package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig *string
	// home是家目录，如果能取得家目录的值，就可以用来做默认值
	if home := homedir.HomeDir(); home != "" {
		// 如果输入了 kubeconfig 参数，该参数的值就是 kubeconfig 文件的绝对路径，
		// 如果没有输入 kubeconfig 参数，就用默认路径 ~/.kube/config
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		// 如果取不到当前用户的家目录，就没办法设置 kubeconfig 的默认目录了，只能从入参中取
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// 从本机加载kubeconfig配置文件，因此第一个参数为空字符串
	restConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	restConfig.APIPath = "api"                            // 参考path : /api/v1/namespaces/{namespace}/pods
	restConfig.GroupVersion = &core_v1.SchemeGroupVersion // pod的 group 是空字符串
	restConfig.NegotiatedSerializer = scheme.Codecs       // 指定序列化工具

	// 根据配置信息构建 restClient 实例
	restClient, err := rest.RESTClientFor(restConfig)
	if err != nil {
		panic(err)
	}

	result := &core_v1.PodList{} // 保存 pod 结果的数据结构实例
	namespace := "kube-system"   // 指定 namespace

	// 设置请求参数，然后发起请求
	// GET 请求
	err = restClient.Get().
		Namespace(namespace).                                                     // 指定 namespace 参考 path : /api/v1/namespaces/{namespace}/pods
		Resource("pods").                                                         // 查找多个pod，参考 path : /api/v1/namespaces/{namespace}/pods
		VersionedParams(&meta_v1.ListOptions{Limit: 100}, scheme.ParameterCodec). // 指定大小限制和序列化工具
		Do(context.TODO()).                                                       // 请求
		Into(result)                                                              // 结果存入 result
	if err != nil {
		panic(err)
	}

	// 每个 pod 都打印 namespace、status.Phase、name 三个字段
	fmt.Printf("namespace\t status\t\t name\n")
	for _, d := range result.Items {
		fmt.Printf("%v\t %v\t %v\n", d.Namespace, d.Status.Phase, d.Name)
	}

	// Output:
	//     namespace        status          name
	//     kube-system      Running         calico-kube-controllers-77c6ddbfb-zvkt6
	//     kube-system      Running         calico-node-4dhcp
	//     kube-system      Running         calico-node-gx7cr
	//     kube-system      Running         calico-node-h8rl6
	//     kube-system      Running         calico-node-plbp2
	//     kube-system      Running         calico-node-txdgt
	//     kube-system      Running         calico-node-wbpxd
	//     kube-system      Running         coredns-6fbf8b5fd4-wcs9x
	//     kube-system      Running         metrics-server-74bdd7786d-s7rdc
	//     kube-system      Running         metrics-server-74bdd7786d-sfr2v
	//     kube-system      Running         metrics-server-74bdd7786d-x8btg

}
