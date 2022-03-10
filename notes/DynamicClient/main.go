package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

/*
- DynamicClient 客户端是一种动态客户端，可以对任意的 Kubernetes 资源进行 RESTful操作，
  包括 CRD 资源。 DynamicClient 内部实现了 Unstructured，用于处理非结构化数据结构（即无
  法提前预知的数据结构），这也是 DynamicClient 能够处理 CRD 资源的关键。
- DynamicClient 不是类型安全的，因此在访问 CRD 自定义资源是要注意，例如，在操作不当时
  可能会导致程序崩溃。 DynamicClient 的处理过程是将 Resource (如 PodList )转换成
  Unstructured 结构类型，Kubernetes 的所有 Resource 都可以转换为该结构类型。处理完后再将
  Unstructured 转换成 PodList。过程类似于 Go 语言的 interface{} 断言转换过程。另外，
  Unstructured 结构类型是通过 map[string]interface{} 转换的。
*/
func main() {
	// 1. 解析 kubeconfig 配置文件
	//    如果能获取到 $HOME 则 kubeconfig 文件的绝对路径为 $HOME/.kube/config
	var kubeConfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeConfig = flag.String("kubeconfig",
			filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// 2. 获取 restConfig
	restConfig, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		panic(err)
	}
	// 3. 获取 RESTClient
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		panic(err)
	}

	// 用于设置请求的资源组，资源版本，资源名称及命名空间
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "pods"}                        // pod
	gvr2 := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"} // deployment
	// List 函数用于获取 Pod 列表 和 Deployment 列表
	unstructuredObj, err := dynamicClient.Resource(gvr).Namespace(core_v1.NamespaceDefault).
		List(context.Background(), meta_v1.ListOptions{Limit: 500})
	if err != nil {
		panic(err)
	}
	// List 函数用于获取 Deployment 列表
	unstructuredObj2, err := dynamicClient.Resource(gvr2).Namespace(core_v1.NamespaceDefault).
		List(context.Background(), meta_v1.ListOptions{Limit: 500})
	if err != nil {
		panic(err)
	}

	podList := &core_v1.PodList{}
	deployList := &apps_v1.Deployment{}

	// 通过 runtime 的函数将 unstructured.UnstructuredList 转换为 PodList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), podList)
	if err != nil {
		panic(err)
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj2.UnstructuredContent(), deployList)
	if err != nil {
		panic(err)
	}

	// 打印
	for _, pod := range podList.Items {
		fmt.Printf("NAMESPACE:%v \t NAME:%v \t STATU:%v\n",
			pod.Namespace, pod.Name, pod.Status.Phase)
	}
	fmt.Println(deployList.Name)
}

func NewDynamicClient() {
}
