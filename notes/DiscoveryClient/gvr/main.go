package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

/*
	从 kubernetes 查询所有的 Group、Version、Resource 信息，在控制台打印出来
*/

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

	// 新建 discoveryClient 实例
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		panic(err.Error())
	}

	// 获取所有分组和资源数据
	APIGroup, APIResourceListSlice, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		panic(err.Error())
	}

	// 先看 Group 信息
	fmt.Printf("APIGroup :\n\n %v\n\n\n\n", APIGroup)
	// APIResourceListSlice 是个切片，里面的每个元素代表一个 GroupVersion 及其资源
	for _, singleAPIResourceList := range APIResourceListSlice {
		// GroupVersion 是个字符串，例如 "apps/v1"
		groupVerionStr := singleAPIResourceList.GroupVersion
		// ParseGroupVersion 方法将字符串转成数据结构
		gv, err := schema.ParseGroupVersion(groupVerionStr)
		if err != nil {
			panic(err.Error())
		}

		fmt.Println("*****************************************************************")
		fmt.Printf("GV string [%v]\nGV struct [%#v]\nresources :\n\n", groupVerionStr, gv)

		// APIResources 字段是个切片，里面是当前 GroupVersion 下的所有资源
		for _, singleAPIResource := range singleAPIResourceList.APIResources {
			fmt.Printf("%v\n", singleAPIResource.Name)
		}
	}
}
