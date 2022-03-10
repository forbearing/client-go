package clientset

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Namespace struct {
	limit   int64
	timeout int64
	ctx     context.Context
	client  *kubernetes.Clientset

	sync.Mutex
}

// new `Namespace` instance from kubeconfig or in-cluster config
func NewNamespace(ctx context.Context, kubeconfig string) (namespace *Namespace, err error) {
	var (
		config *rest.Config
		client *kubernetes.Clientset
	)
	namespace = &Namespace{}

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

	namespace.limit = 100
	namespace.timeout = 10
	namespace.ctx = ctx
	namespace.client = client

	return
}
func (n *Namespace) SetLimit(limit int64) {
	n.Lock()
	defer n.Unlock()
	n.limit = limit
}
func (n *Namespace) SetTimeout(timeout int64) {
	n.Lock()
	defer n.Unlock()
	n.timeout = timeout
}

// create namespace from file
func (n *Namespace) Create(filepath string) (namespace *corev1.Namespace, err error) {
	var (
		namespaceYaml []byte
		namespaceJson []byte
	)
	namespace = &corev1.Namespace{}
	if namespaceYaml, err = ioutil.ReadFile(filepath); err != nil {
		return
	}
	if namespaceJson, err = yaml.ToJSON(namespaceYaml); err != nil {
		return
	}
	if err = json.Unmarshal(namespaceJson, namespace); err != nil {
		return
	}
	namespace, err = n.client.CoreV1().Namespaces().Create(n.ctx, namespace, metav1.CreateOptions{})
	return
}

// update namespace from file
func (n *Namespace) Update(filepath string) (namespace *corev1.Namespace, err error) {
	var (
		namespaceYaml []byte
		namespaceJson []byte
	)
	namespace = &corev1.Namespace{}
	if namespaceYaml, err = ioutil.ReadFile(filepath); err != nil {
		return
	}
	if namespaceJson, err = yaml.ToJSON(namespaceYaml); err != nil {
		return
	}
	if err = json.Unmarshal(namespaceJson, namespace); err != nil {
		return
	}
	namespace, err = n.client.CoreV1().Namespaces().Update(n.ctx, namespace, metav1.UpdateOptions{})
	return
}

// apply namespace from file
func (n *Namespace) Apply(filepath string) (namespace *corev1.Namespace, err error) {
	namespace, err = n.Create(filepath)
	if errors.IsAlreadyExists(err) {
		namespace, err = n.Update(filepath)
	}
	return
}

// delete namespace from file
func (n *Namespace) Delete(name string, forceDelete bool) (err error) {
	var gracePeriodSeconds int64
	if forceDelete {
		gracePeriodSeconds = 0
		err = n.client.CoreV1().Namespaces().Delete(n.ctx,
			name, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds})
		return
	}
	err = n.client.CoreV1().Namespaces().Delete(n.ctx, name, metav1.DeleteOptions{})
	return
}

// get namespace by name
func (n *Namespace) Get(name string) (namespace *corev1.Namespace, err error) {
	namespace, err = n.client.CoreV1().Namespaces().Get(n.ctx, name, metav1.GetOptions{})
	return
}

// list namespace by labelSelector
func (n *Namespace) List(labelSelector string) (namespaceList *corev1.NamespaceList, err error) {
	namespaceList, err = n.client.CoreV1().Namespaces().List(n.ctx,
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &n.timeout, Limit: n.limit})
	return
}

// watch pods by labelSelector
func (n *Namespace) Watch(labelSelector string, addFunc, modifyFunc, deleteFunc func()) (err error) {
	var (
		watcher       watch.Interface
		namespaceList *corev1.NamespaceList
		timeout       = int64(0)
		isExist       bool
	)
	for {
		if watcher, err = n.client.CoreV1().Namespaces().Watch(n.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout, Limit: n.limit}); err != nil {
			logrus.Error(err)
			return
		}
		if namespaceList, err = n.List(labelSelector); err != nil {
			logrus.Error(err)
			return
		}
		if len(namespaceList.Items) == 0 {
			isExist = false // namespace not exist
		} else {
			isExist = true // namespace exist
		}
		for {
			event, isOpen := <-watcher.ResultChan()
			if isOpen {
				switch event.Type {
				case watch.Added:
					if !isExist {
						addFunc()
					}
					isExist = true
				case watch.Modified:
					modifyFunc()
					isExist = true
				case watch.Deleted:
					deleteFunc()
					isExist = false
				default:
				}
			} else {
				// If event channel is closed, it means the server has closed the connection
				logrus.Info("reconnect to kube-apiserver")
				break
			}
		}
	}
}
