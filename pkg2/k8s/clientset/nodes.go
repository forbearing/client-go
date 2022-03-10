package clientset

import (
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Node struct {
	limit   int64
	timeout int64
	ctx     context.Context
	client  *kubernetes.Clientset

	sync.Mutex
}

func NewNode(ctx context.Context, kubeconfig string) (node *Node, err error) {
	var (
		config *rest.Config
		client *kubernetes.Clientset
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
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return
		}
	}

	// create the clientset
	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	node.limit = 100
	node.timeout = 10
	node.ctx = ctx
	node.client = client

	return
}
func (n *Node) SetTimeout(timeout int64) {
	n.Lock()
	defer n.Unlock()
	n.timeout = timeout
}
func (n *Node) SetLimit(limit int64) {
	n.Lock()
	defer n.Unlock()
	n.limit = limit
}

func (n *Node) Get(name string) (node *corev1.Node, err error) {
	node, err = n.client.CoreV1().Nodes().Get(n.ctx, name, metav1.GetOptions{})
	return
}

func (n *Node) List(labelSelector string) (nodeList *corev1.NodeList, err error) {
	nodeList, err = n.client.CoreV1().Nodes().List(n.ctx,
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &n.timeout, Limit: n.limit})
	return
}
