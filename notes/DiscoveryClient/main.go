package main

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/clientcmd"
)

/*
- DiscoveryClient 是发现客户端，主要用于发现 Kubenetes API Server 所支持的资源组、
  资源版本、资源信息。 kubectl 的 api-versions 和 api-resources 命令输出也是通过
  DiscoveryClient 实现的。其同样是在 RESTClient 的基础上进行的封装。DiscoveryClient
  还可以将资源组、资源版本、资源信息等存储在本地，用于本地缓存，减轻对 kubernetes api sever
  的访问压力，缓存信息默认存储在：~/.kube/cache 和 ~/.kube/http-cache 下。
*/

func main() {
	discoveryClient, err := NewDiscoveryClient()
	if err != nil {
		panic(err)
	}

	_, APIResourceList, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		panic(err)
	}

	// list.APIResources 包含了 API resource
	// list.GroupVersion 包含了 API resource 对应的 Group 和 Version
	for _, list := range APIResourceList {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			panic(err)
		}

		fmt.Println("name | group | version")
		for _, resource := range list.APIResources {
			// fmt.Printf("name:%v,group:%v,version:%v\n", resource.Name, gv.Group, gv.Version)
			fmt.Println(resource.Name, gv.Group, gv.Version)
		}
	}

}

func NewDiscoveryClient() (*discovery.DiscoveryClient, error) {
	var (
		discoveryClient *discovery.DiscoveryClient
		err             error
	)
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		panic(err)
	}

	discoveryClient, err = discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return discoveryClient, err
	}

	return discoveryClient, nil
}
