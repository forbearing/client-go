package k8s

// TODO:
// 1. GetNonTerminatedPods 方法有问题,需要修改
//    参考: https://github.com/pytimer/k8sutil/blob/main/node/node.go
// 2. GetNodeInfo 需要判断两种 role
import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	_ "k8s.io/metrics/pkg/apis/metrics"
	_ "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Node struct {
	kubeconfig string

	ctx             context.Context
	config          *rest.Config
	restClient      *rest.RESTClient
	clientset       *kubernetes.Clientset
	dynamicClient   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient
	informerFactory informers.SharedInformerFactory

	Options *HandlerOptions

	sync.Mutex
}

// new a node handler from kubeconfig or in-cluster config
func NewNode(ctx context.Context, kubeconfig string) (node *Node, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	node = &Node{}

	// create rest config
	if len(kubeconfig) != 0 {
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return
		}
	} else {
		// create the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return
		}
	}

	// setup APIPath, GroupVersion and NegotiatedSerializer before initializing a RESTClient
	config.APIPath = "api"
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs
	// create a RESTClient for the given config
	restClient, err = rest.RESTClientFor(config)
	if err != nil {
		return
	}
	// create a Clientset for the given config
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return
	}
	// create a dynamic client for the given config
	dynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return
	}
	// create a DiscoveryClient for the given config
	discoveryClient, err = discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return
	}
	// create a sharedInformerFactory for all namespaces.
	informerFactory = informers.NewSharedInformerFactory(clientset, time.Minute)

	node.kubeconfig = kubeconfig
	node.ctx = ctx
	node.config = config
	node.restClient = restClient
	node.clientset = clientset
	node.dynamicClient = dynamicClient
	node.discoveryClient = discoveryClient
	node.informerFactory = informerFactory
	node.Options = &HandlerOptions{}

	return
}
func (in *Node) DeepCopy() *Node {
	out := new(Node)

	out.kubeconfig = in.kubeconfig

	out.ctx = in.ctx
	out.config = in.config
	out.restClient = in.restClient
	out.clientset = in.clientset
	out.dynamicClient = in.dynamicClient
	out.discoveryClient = in.discoveryClient

	out.Options = &HandlerOptions{}
	out.Options.ListOptions = *in.Options.ListOptions.DeepCopy()
	out.Options.GetOptions = *in.Options.GetOptions.DeepCopy()
	out.Options.CreateOptions = *in.Options.CreateOptions.DeepCopy()
	out.Options.UpdateOptions = *in.Options.UpdateOptions.DeepCopy()
	out.Options.PatchOptions = *in.Options.PatchOptions.DeepCopy()
	out.Options.ApplyOptions = *in.Options.ApplyOptions.DeepCopy()

	return out
}
func (n *Node) WithDryRun() *Node {
	node := n.DeepCopy()
	node.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	node.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	node.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	node.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	node.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return node
}
func (n *Node) SetTimeout(timeout int64) {
	n.Lock()
	defer n.Unlock()
	n.Options.ListOptions.TimeoutSeconds = &timeout
}
func (n *Node) SetLimit(limit int64) {
	n.Lock()
	defer n.Unlock()
	n.Options.ListOptions.Limit = limit
}

func (n *Node) GetByName(name string) (*corev1.Node, error) {
	return n.clientset.CoreV1().Nodes().Get(n.ctx, name, n.Options.GetOptions)
}

func (n *Node) Get(name string) (*corev1.Node, error) {
	return n.GetByName(name)
}

// ListByLabel list nodes by labels
func (n *Node) ListByLabel(labels string) (*corev1.NodeList, error) {
	listOptions := n.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return n.clientset.CoreV1().Nodes().List(n.ctx, *listOptions)
}

// List list nodes by labels, alias to "ListByLabel"
func (n *Node) List(labels string) (*corev1.NodeList, error) {
	return n.ListByLabel(labels)
}

// ListAll list all nodes
func (n *Node) ListAll() (*corev1.NodeList, error) {
	return n.ListByLabel("")
}

// check if the node status is ready
func (n *Node) IsReady(name string) bool {
	// get *corev1.Node
	node, err := n.Get(name)
	if err != nil {
		return false
	}
	for _, cond := range node.Status.Conditions {
		if cond.Status == corev1.ConditionTrue && cond.Type == corev1.NodeReady {
			return true
		}
	}
	return false
}

// check if the node is master
func (n *Node) IsMaster(name string) bool {
	roles := n.GetRoles(name)
	for _, role := range roles {
		if strings.ToLower(role) == NodeRoleMaster {
			return true
		}
	}
	return false
}

// check if the node is control-plane
func (n *Node) IsControlPlane(name string) bool {
	roles := n.GetRoles(name)
	for _, role := range roles {
		if strings.ToLower(role) == NodeRoleControlPlane {
			return true
		}
	}
	return false
}

// get the node status
func (n *Node) GetStatus(name string) *NodeStatus {
	nodeStatus := &NodeStatus{
		Message: "Unknow",
		Reason:  "Unknow",
		Status:  string(corev1.ConditionUnknown),
	}

	// get *corev1.Node
	node, err := n.Get(name)
	if err != nil {
		return nodeStatus
	}

	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			nodeStatus = &NodeStatus{
				Message: cond.Message,
				Reason:  cond.Reason,
				Status:  string(cond.Status),
			}
		}
	}

	return nodeStatus
}

// GetRoles returns the roles of a given node.
// The roles are determined by looking for:
//   node-role.kubernetes.io/<role>=""
//   kubernetes.io/role="<role>"
func (n *Node) GetRoles(name string) []string {
	roles := sets.NewString()

	// get *corev1.Node
	node, err := n.Get(name)
	if err != nil {
		return roles.List()
	}

	for label, value := range node.Labels {
		switch {
		case strings.HasPrefix(label, LabelNodeRolePrefix):
			if role := strings.TrimPrefix(label, LabelNodeRolePrefix); len(role) > 0 {
				roles.Insert(role)
			}
		case label == LabelNodeRole && len(value) > 0:
			roles.Insert(value)
		}
	}

	return roles.List()
}

// get all pods in the node
func (n *Node) GetPods(name string) (*corev1.PodList, error) {
	// ParseSelector takes a string representing a selector and returns an
	// object suitable for matching, or an error.
	fieldSelector, err := fields.ParseSelector(fmt.Sprintf("spec.nodeName=%s", name))
	if err != nil {
		return nil, err
	}

	podHandler, err := NewPod(n.ctx, "", n.kubeconfig)
	if err != nil {
		return nil, err
	}
	podHandler.Options.ListOptions = metav1.ListOptions{FieldSelector: fieldSelector.String()}
	//podHandler.SetNamespace(metav1.NamespaceAll)
	//return podHandler.List("")
	return podHandler.WithNamespace(metav1.NamespaceAll).List("")
}

// get not terminated pod in the node.
func (n *Node) GetNonTerminatedPods(name string) (*corev1.PodList, error) {
	// PodSucceeded 表示 containers 成功退出, pod 终止
	// PodSucceeded 表示 containers 失败退出, pod 也终止
	// PodPending, PodRunning, PodUnknown 都表示 pod 正在运行
	selector := fmt.Sprintf("spec.nodeName=%s,status.phase!=%s,status.phase!=%s",
		name, string(corev1.PodSucceeded), string(corev1.PodFailed))
	// ParseSelector takes a string representing a selector and returns an
	// object suitable for matching, or an error.
	fieldSelector, err := fields.ParseSelector(selector)
	if err != nil {
		return nil, err
	}
	podHandler, err := NewPod(n.ctx, "", n.kubeconfig)
	if err != nil {
		return nil, err
	}
	podHandler.Options.ListOptions = metav1.ListOptions{FieldSelector: fieldSelector.String()}
	return podHandler.WithNamespace(metav1.NamespaceAll).List("")
}

// get the node ip
func (n *Node) GetIP(name string) (ip string, err error) {
	node, err := n.Get(name)
	if err != nil {
		return
	}
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			ip = address.Address
		}
	}
	return
}

// get the node hostname
func (n *Node) GetHostname(name string) (hostname string, err error) {
	node, err := n.Get(name)
	if err != nil {
		return
	}
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeHostName {
			hostname = address.Address
		}
	}
	return
}

// get the node podCIDR
func (n *Node) GetCIDR(name string) (string, error) {
	node, err := n.Get(name)
	if err != nil {
		return "", err
	}
	return node.Spec.PodCIDR, nil
}

// get the node podCIDRs
func (n *Node) GetCIDRs(name string) ([]string, error) {
	node, err := n.Get(name)
	if err != nil {
		return nil, err
	}
	return node.Spec.PodCIDRs, nil
}

// get all master node info
func (n *Node) GetMasterInfo() ([]NodeInfo, error) {
	var nodeInfo NodeInfo
	var nodeInfoList []NodeInfo

	masterNodes, err := n.List(LabelNodeRolePrefix + "master")
	if err != nil {
		return nil, err
	}
	for _, node := range masterNodes.Items {
		nodeInfo.Hostname = node.ObjectMeta.Name
		nodeInfo.IPAddress, _ = n.GetIP(nodeInfo.Hostname)
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
		nodeInfoList = append(nodeInfoList, nodeInfo)
	}

	return nodeInfoList, nil
}

// get all worker node info
func (n *Node) GetWorkerInfo() ([]NodeInfo, error) {
	var nodeInfo NodeInfo
	var nodeInfoList []NodeInfo

	workerNodes, err := n.List("!" + LabelNodeRolePrefix + "master")
	if err != nil {
		return nil, err
	}
	for _, node := range workerNodes.Items {
		nodeInfo.Hostname = node.ObjectMeta.Name
		nodeInfo.IPAddress, _ = n.GetIP(nodeInfo.Hostname)
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
		nodeInfoList = append(nodeInfoList, nodeInfo)
	}
	return nodeInfoList, nil
}

// get all k8s node info
func (n *Node) GetAllInfo() ([]NodeInfo, error) {
	var nodeInfoList []NodeInfo
	masterInfo, err := n.GetMasterInfo()
	if err != nil {
		return nil, err
	}
	workerInfo, err := n.GetWorkerInfo()
	if err != nil {
		return nil, err
	}

	for _, info := range masterInfo {
		nodeInfoList = append(nodeInfoList, info)
	}
	for _, info := range workerInfo {
		nodeInfoList = append(nodeInfoList, info)
	}

	return nodeInfoList, nil
}
