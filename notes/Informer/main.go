package main

import (
	"log"
	"os"
	"time"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

/*
- Reflector：用于监控(Watch)指定的Kubernetes资源，当监控的资源发生变化时触发响应的变更
  事件（例如 Added 事件、Updated 事件和 Deleted 事件），并将其资源对象存放到本地缓存
  DeltaFIFO中。
- DeltaFIFO：可以分开理解。FIFO 是一个先进先出的队列，拥有队列操作的基本方法（Add，Update，
  Delete，List，Pop，Close等）；Delta 是一个资源对象存储，可以保存资源对象的操作类型
  （Added、Updated、Deleted、Sync等）
- Indexer：是 client-go 用来存储资源对象并自带索引功能的本地存储，Reflector 从 DeltaFIFO
  中将消费出来的资源对象存储至 Indexer。Indexer 与 Etcd 集群中的数据完全一致。client-go
  可以很方便的从本地存储中读取相应的资源对象数据，无须每次从远程Etcd集群读取，减轻了
  api-server 及 Etcd 集群的压力。


- kubernetes 上的每一个资源都实现了 Informer 机制，每一个 Informer 上都会实现 Informer 和 Lister 方法
- Informer 也被称为 Shared Informer，它是可以共享使用的。若同一资源的 Informer 被实例化了
  多次，每个 Informer 使用一个 Reflector，那么会运行过多的相同 ListAndWatch ，太多重复的
  序列化和反序列化操作会导致 api-server 负载过重。
- Shared Informer 可以使同一个资源对象共享一个 Reflector，这样可以节约很多资源；Shared Infor
  定义了一个 map 数据结构，通过 map 数据结构实现共享 Informer 机制。

*/

func main() {
	// 获取 kubeconfig 文件的绝对路径
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"
	// 1. 获取 restConfig 文件
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		panic(err)
	}
	// 2. 创建 clientset 对象
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		panic(err)
	}
	// 创建 stop channel 对象，用于在程序进程退出之前通知 Informer 退出，
	// 因为 Informer 是一个持久运行的 goroutine
	stopCh := make(chan struct{})
	defer close(stopCh)

	// 实例化 ShareInformer 对象，一个参数是 clientset, 另一个是 time.Minute 用于设置多久进行一次 resync(重新同步)
	// resync 会周期性的执行 List 操作，将所有的资源存放在 Informer Store 中，如果参数为0,则禁止 resync 操作
	sharedInformers := informers.NewSharedInformerFactory(clientset, time.Minute)
	// 得到具体的 Pod 资源的 informer 对象
	podInformer := sharedInformers.Core().V1().Pods().Informer()
	// 为 Pod 资源添加资源事件回调方法，支持 AddFunc、UpdateFunc 及 DeleteFunc
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// 在正常情况下，kubernetes 其他组件在使用 Informer 机制时触发资源事件回调方法，
			// 将资源对象推送到 WorkQueue 或其他队列中，
			// 这里是直接输出触发的资源事件
			myObj := obj.(meta_v1.Object)
			log.Printf("New Pod Added to Store: %s", myObj.GetName())
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oObj := oldObj.(meta_v1.Object)
			nObj := newObj.(meta_v1.Object)
			log.Printf("%s Pod Updated to %s", oObj.GetName(), nObj.GetName())
		},
		DeleteFunc: func(obj interface{}) {
			myObj := obj.(meta_v1.Object)
			log.Printf("Pod Deleted from Store: %s", myObj.GetName())
		},
	})
	// 通过 Run 函数运行当前的 Informer，内部为 Pod 资源类型创建的 Informer
	podInformer.Run(stopCh)
}
