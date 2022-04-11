package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig *string

	// home是家目录，如果能取得家目录的值，就可以用来做默认值
	if home := homedir.HomeDir(); home != "" {
		// 如果输入了kubeconfig参数，该参数的值就是kubeconfig文件的绝对路径，
		// 如果没有输入kubeconfig参数，就用默认路径~/.kube/config
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		// 如果取不到当前用户的家目录，就没办法设置kubeconfig的默认目录了，只能从入参中取
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.Parse()

	// 从本机加载 kubeconfig 配置文件，因此第一个参数为空字符串
	restConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		panic(err.Error())
	}

	// dynamicClient 的唯一关联方法所需的入参
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "pods"}

	// 使用 dynamicClient 的查询列表方法，查询指定 namespace 下的所有 pod，
	// 注意此方法返回的数据结构类型是 UnstructuredList
	unstructObj, err := dynamicClient.
		Resource(gvr).
		Namespace("kube-system").
		List(context.TODO(), meta_v1.ListOptions{Limit: 100})
	if err != nil {
		panic(err.Error())
	}

	// 实例化一个 PodList 数据结构，用于接收 unstructObj 转换后的结果
	podList := &core_v1.PodList{}

	// 转换
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructObj.UnstructuredContent(), podList)
	if err != nil {
		panic(err.Error())
	}

	// 每个 pod 都打印 namespace、status.Phase、name三个字段
	fmt.Printf("namespace\t status\t\t name\n")
	for _, d := range podList.Items {
		fmt.Printf("%v\t %v\t %v\n",
			d.Namespace,
			d.Status.Phase,
			d.Name)
	}
}
