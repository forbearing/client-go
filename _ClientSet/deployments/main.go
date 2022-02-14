package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/utils/pointer"
)

/*
如果 -operate 选项值为 create（默认选项），则创建如下 k8s 资源
	1. 创建 namespace test
	2. 创建 deployment nginx
	3. 创建 service nginx
如果 -operate 选项值为 clean，则删除
	1. 删除 service
	2. 删除 deployment
	3. 删除 namespace
*/

const (
	NAMESPACE  = "test"
	DEPLOYMENT = "nginx"
	SERVICE    = "nginx"
)

func main() {
	var kubeconfig *string
	var operate *string

	// home是家目录，如果能取得家目录的值，就可以用来做默认值
	if home := homedir.HomeDir(); home != "" {
		// 如果输入了kubeconfig参数，该参数的值就是kubeconfig文件的绝对路径，
		// 如果没有输入kubeconfig参数，就用默认路径~/.kube/config
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		// 如果取不到当前用户的家目录，就没办法设置kubeconfig的默认目录了，只能从入参中取
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	// 获取用户输入的操作类型，默认是create，还可以输入clean，用于清理所有资源
	operate = flag.String("operate", "create", "operate type : create or clean")
	flag.Parse()

	// 从本机加载kubeconfig配置文件，因此第一个参数为空字符串
	restConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// 实例化clientset 对象
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("operation is %v\n", *operate)

	if *operate == "clean" {
		clean(clientset)
	} else {
		createNamespace(clientset)
		createService(clientset)
		createDeployment(clientset)
	}
}

// 删除 service、deployment、namespace
func clean(clientset *kubernetes.Clientset) {
	emptyDeleteOptions := meta_v1.DeleteOptions{}

	// 删除 service
	if err := clientset.CoreV1().Services(NAMESPACE).Delete(context.TODO(),
		SERVICE, emptyDeleteOptions); err != nil {
		panic(err.Error())
	}
	// 删除 deployment
	if err := clientset.AppsV1().Deployments(NAMESPACE).Delete(context.TODO(),
		DEPLOYMENT, emptyDeleteOptions); err != nil {
		panic(err.Error())
	}
	// 删除 namespace
	if err := clientset.CoreV1().Namespaces().Delete(context.TODO(),
		NAMESPACE, emptyDeleteOptions); err != nil {
		panic(err.Error())
	}
}

func createNamespace(clientset *kubernetes.Clientset) {
	namespaceClient := clientset.CoreV1().Namespaces()
	namespace := &core_v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{Name: NAMESPACE},
	}

	result, err := namespaceClient.Create(context.TODO(), namespace, meta_v1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Create namespace: %s\n", result.GetName())
}

func createService(clientset *kubernetes.Clientset) {
	// 得到 service 的客户端
	serviceClient := clientset.CoreV1().Services(NAMESPACE)
	// 实例化一个数据结构
	service := &core_v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{Name: SERVICE},
		Spec: core_v1.ServiceSpec{
			Ports: []core_v1.ServicePort{{
				Name:     "http",
				Port:     80,
				NodePort: 30080,
			}},
			Selector: map[string]string{
				"app": "nginx",
			},
			Type: core_v1.ServiceTypeNodePort,
		},
	}

	result, err := serviceClient.Create(context.TODO(), service, meta_v1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Create Service %s\n", result.GetName())
}

func createDeployment(clientset *kubernetes.Clientset) {
	// 得到 deployment 客户端
	deploymentClient := clientset.AppsV1().Deployments(NAMESPACE)
	// 实例化一个数据结构
	deployment := &apps_v1.Deployment{
		ObjectMeta: meta_v1.ObjectMeta{Name: DEPLOYMENT},
		Spec: apps_v1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(2),
			Selector: &meta_v1.LabelSelector{MatchLabels: map[string]string{"app": "nginx"}},
			Template: core_v1.PodTemplateSpec{
				ObjectMeta: meta_v1.ObjectMeta{Labels: map[string]string{"app": "nginx"}},
				Spec: core_v1.PodSpec{
					Containers: []core_v1.Container{{
						Name:            "nginx",
						Image:           "nginx",
						ImagePullPolicy: "IfNotPresent",
						Ports: []core_v1.ContainerPort{{
							Name:          "http",
							Protocol:      core_v1.ProtocolTCP,
							ContainerPort: 80,
						}},
					}},
				},
			},
		},
	}

	result, err := deploymentClient.Create(context.TODO(), deployment, meta_v1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Create deployment: %s\n", result.GetName())
}
