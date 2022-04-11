package k8s

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	//yamlv3 "gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
)

func watchHandler(eventChannel <-chan watch.Event, mutex *sync.Mutex) {
	for { // {{{
		event, open := <-eventChannel
		if open {
			switch event.Type {
			case watch.Added:
				logrus.Info("add")
			case watch.Modified:
				logrus.Info("modified")
			case watch.Deleted:
				logrus.Info("deleted")
			case watch.Bookmark:
				logrus.Info("bookmark")
			case watch.Error:
				logrus.Error("error")
			default: // do nothing
			}
		} else {
			// If eventChannel is closed, it means the server has closed the connection
			return
		}
	}
} // }}}
func createDeployment(clientset *kubernetes.Clientset) {
	var ( // {{{
		NAMESPACE  = "default"
		DEPLOYMENT = "nginx"
	)
	// 得到 deployment 客户端
	deploymentClient := clientset.AppsV1().Deployments(NAMESPACE)
	// 实例化一个数据结构
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: DEPLOYMENT},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(2),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "nginx"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "nginx"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "nginx",
						Image:           "nginx",
						ImagePullPolicy: "IfNotPresent",
						Ports: []corev1.ContainerPort{{
							Name:          "http",
							Protocol:      corev1.ProtocolTCP,
							ContainerPort: 80,
						}},
					}},
				},
			},
		},
	}

	result, err := deploymentClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Create deployment: %s\n", result.GetName())
} // }}}

func GetNodeInfo(ctx context.Context, kubeconfig string) (nodeInfoMap map[string]NodeInfo, err error) {
	var (
		nodeObj  *Node
		nodeList *corev1.NodeList
		nodeInfo NodeInfo
	)
	// map 使用之前一定要初始化一下
	nodeInfoMap = make(map[string]NodeInfo)
	// 获取一个自定义的 Node 对象
	if nodeObj, err = NewNode(ctx, kubeconfig); err != nil {
		return
	}
	// 通过 Node 对象获取到 corev1.NodeList 对象
	// 通过 "node-role.kubernetes.io/master" 获取到的是所有的 master 节点列表
	if nodeList, err = nodeObj.List("node-role.kubernetes.io/master"); err != nil {
		return
	}
	for _, node := range nodeList.Items {
		nodeInfo.Hostname = node.ObjectMeta.Name
		nodeInfo.AllocatableCpu = node.Status.Allocatable.Cpu().String()
		nodeInfo.AllocatableMemory = node.Status.Allocatable.Memory().String()
		nodeInfo.AllocatableStorage = node.Status.Allocatable.StorageEphemeral().String()
		nodeInfo.Architecture = node.Status.NodeInfo.Architecture
		nodeInfo.TotalCpu = node.Status.Capacity.Cpu().String()
		nodeInfo.TotalMemory = node.Status.Capacity.Memory().String()
		nodeInfo.TotalStorage = node.Status.Capacity.StorageEphemeral().String()
		nodeInfo.BootID = node.Status.NodeInfo.BootID
		nodeInfo.ContainerRuntimeVersion = node.Status.NodeInfo.ContainerRuntimeVersion
		nodeInfo.KernelVersion = node.Status.NodeInfo.KernelVersion
		nodeInfo.KubeProxyVersion = node.Status.NodeInfo.KubeProxyVersion
		nodeInfo.KubeletVersion = node.Status.NodeInfo.KubeletVersion
		nodeInfo.MachineID = node.Status.NodeInfo.MachineID
		nodeInfo.OperatingSystem = node.Status.NodeInfo.OperatingSystem
		nodeInfo.OSImage = node.Status.NodeInfo.OSImage
		nodeInfo.SystemUUID = node.Status.NodeInfo.SystemUUID
		// map 的 key 就是 node.ObjectMeta.Name, 即 k8s 节点的 ip 地址
		nodeInfoMap[nodeInfo.Hostname] = nodeInfo
	}
	// 通过 "!node-role.kubernetes.io/master" 获取到的是所有的非 master 节点列表, 即所有的 worker 节点列表
	if nodeList, err = nodeObj.List("!node-role.kubernetes.io/master"); err != nil {
		return
	}
	for _, node := range nodeList.Items {
		nodeInfo.Hostname = node.ObjectMeta.Name
		nodeInfo.AllocatableCpu = node.Status.Allocatable.Cpu().String()
		nodeInfo.AllocatableMemory = node.Status.Allocatable.Memory().String()
		nodeInfo.AllocatableStorage = node.Status.Allocatable.StorageEphemeral().String()
		nodeInfo.TotalCpu = node.Status.Capacity.Cpu().String()
		nodeInfo.TotalMemory = node.Status.Capacity.Memory().String()
		nodeInfo.TotalStorage = node.Status.Capacity.StorageEphemeral().String()
		nodeInfo.Architecture = node.Status.NodeInfo.Architecture
		nodeInfo.BootID = node.Status.NodeInfo.BootID
		nodeInfo.ContainerRuntimeVersion = node.Status.NodeInfo.ContainerRuntimeVersion
		nodeInfo.KernelVersion = node.Status.NodeInfo.KernelVersion
		nodeInfo.KubeProxyVersion = node.Status.NodeInfo.KubeProxyVersion
		nodeInfo.KubeletVersion = node.Status.NodeInfo.KubeletVersion
		nodeInfo.MachineID = node.Status.NodeInfo.MachineID
		nodeInfo.OperatingSystem = node.Status.NodeInfo.OperatingSystem
		nodeInfo.OSImage = node.Status.NodeInfo.OSImage
		nodeInfo.SystemUUID = node.Status.NodeInfo.SystemUUID
		// map 的 key 就是 node.ObjectMeta.Name, 即 k8s 节点的 ip 地址
		nodeInfoMap[nodeInfo.Hostname] = nodeInfo
	}

	return
}

/*
1. 重复 apply 一个 pvc 会失败,因为 pvc.spec.volumeName 绑定的 pv 不允许修改
*/
