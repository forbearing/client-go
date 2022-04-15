package k8s

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Namespace struct {
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

// new a namespace handler from kubeconfig or in-cluster config
func NewNamespace(ctx context.Context, kubeconfig string) (namespace *Namespace, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
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

	namespace.kubeconfig = kubeconfig
	namespace.ctx = ctx
	namespace.config = config
	namespace.restClient = restClient
	namespace.clientset = clientset
	namespace.dynamicClient = dynamicClient
	namespace.discoveryClient = discoveryClient
	namespace.informerFactory = informerFactory
	namespace.Options = &HandlerOptions{}

	return
}
func (n *Namespace) SetTimeout(timeout int64) {
	n.Lock()
	defer n.Unlock()
	n.Options.ListOptions.TimeoutSeconds = &timeout
}
func (in *Namespace) DeepCopy() *Namespace {
	out := new(Namespace)

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
func (n *Namespace) WithDryRun() *Namespace {
	ns := n.DeepCopy()
	ns.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	ns.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	ns.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	ns.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	ns.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return ns
}
func (n *Namespace) SetLimit(limit int64) {
	n.Lock()
	defer n.Unlock()
	n.Options.ListOptions.Limit = limit
}
func (n *Namespace) SetForceDelete(force bool) {
	n.Lock()
	defer n.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		n.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		n.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create namespace from map[string]interface{}
func (n *Namespace) CreateFromRaw(raw map[string]interface{}) (*corev1.Namespace, error) {
	namespace := &corev1.Namespace{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, namespace)
	if err != nil {
		return nil, err
	}

	return n.clientset.CoreV1().Namespaces().Create(n.ctx, namespace, n.Options.CreateOptions)
}

// CreateFromBytes create namespace from bytes
func (n *Namespace) CreateFromBytes(data []byte) (*corev1.Namespace, error) {
	nsJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	ns := &corev1.Namespace{}
	err = json.Unmarshal(nsJson, ns)
	if err != nil {
		return nil, err
	}

	return n.clientset.CoreV1().Namespaces().Create(n.ctx, ns, n.Options.CreateOptions)
}

// CreateFromFile create namespace from yaml file
func (n *Namespace) CreateFromFile(path string) (*corev1.Namespace, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return n.CreateFromBytes(data)
}

// Create create namespace from yaml file, alias to "CreateFromFile"
func (n *Namespace) Create(path string) (*corev1.Namespace, error) {
	return n.CreateFromFile(path)
}

// UpdateFromRaw update namespace from map[string]interface{}
func (n *Namespace) UpdateFromRaw(raw map[string]interface{}) (*corev1.Namespace, error) {
	namespace := &corev1.Namespace{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, namespace)
	if err != nil {
		return nil, err
	}

	return n.clientset.CoreV1().Namespaces().Update(n.ctx, namespace, n.Options.UpdateOptions)
}

// UpdateFromBytes update namespace from bytes
func (n *Namespace) UpdateFromBytes(data []byte) (*corev1.Namespace, error) {
	nsJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	ns := &corev1.Namespace{}
	err = json.Unmarshal(nsJson, ns)
	if err != nil {
		return nil, err
	}

	return n.clientset.CoreV1().Namespaces().Update(n.ctx, ns, n.Options.UpdateOptions)
}

// UpdateFromFile update namespace from yaml file
func (n *Namespace) UpdateFromFile(path string) (*corev1.Namespace, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return n.UpdateFromBytes(data)
}

// Update update namespace from yaml file, alias to "UpdateFromFile"
func (n *Namespace) Update(path string) (*corev1.Namespace, error) {
	return n.UpdateFromFile(path)
}

// ApplyFromRaw apply namespace from map[string]interface{}
func (n *Namespace) ApplyFromRaw(raw map[string]interface{}) (*corev1.Namespace, error) {
	namespace := &corev1.Namespace{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, namespace)
	if err != nil {
		return nil, err
	}

	namespace, err = n.clientset.CoreV1().Namespaces().Create(n.ctx, namespace, n.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		namespace, err = n.clientset.CoreV1().Namespaces().Update(n.ctx, namespace, n.Options.UpdateOptions)
	}
	return namespace, err
}

// ApplyFromBytes apply namespace from bytes
func (n *Namespace) ApplyFromBytes(data []byte) (namespace *corev1.Namespace, err error) {
	namespace, err = n.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		namespace, err = n.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply namespace from yaml file
func (n *Namespace) ApplyFromFile(path string) (namespace *corev1.Namespace, err error) {
	namespace, err = n.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		namespace, err = n.UpdateFromFile(path)
	}
	return
}

// Apply apply namespace from yaml file, alias to "ApplyFromFile"
func (n *Namespace) Apply(path string) (*corev1.Namespace, error) {
	return n.ApplyFromFile(path)
}

// DeleteFromBytes delete namespace from bytes
func (n *Namespace) DeleteFromBytes(data []byte) error {
	nsJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	ns := &corev1.Namespace{}
	err = json.Unmarshal(nsJson, ns)
	if err != nil {
		return err
	}

	return n.DeleteByName(ns.Name)
}

// DeleteFromFile delete namespace from yaml file
func (n *Namespace) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return n.DeleteFromBytes(data)
}

// DeleteByName delete namespace by name
func (n *Namespace) DeleteByName(name string) error {
	return n.clientset.CoreV1().Namespaces().Delete(n.ctx, name, n.Options.DeleteOptions)
}

// Delete delete namespace by name, alias to "DeleteByName"
func (n *Namespace) Delete(name string) error {
	return n.DeleteByName(name)
}

// GetFromBytes get namespace from bytes
func (n *Namespace) GetFromBytes(data []byte) (*corev1.Namespace, error) {
	nsJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	ns := &corev1.Namespace{}
	if err = json.Unmarshal(nsJson, ns); err != nil {
		return nil, err
	}

	return n.GetByName(ns.Name)
}

// GetFromFile get namespace from yaml file
func (n *Namespace) GetFromFile(path string) (*corev1.Namespace, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return n.GetFromBytes(data)
}

// GetByName get namespace by name
func (n *Namespace) GetByName(name string) (*corev1.Namespace, error) {
	return n.clientset.CoreV1().Namespaces().Get(n.ctx, name, n.Options.GetOptions)
}

// Get get namespace by name, alias to "GetByName"
func (n *Namespace) Get(name string) (*corev1.Namespace, error) {
	return n.GetByName(name)
}

// ListByLabel list namespaces by labels
func (n *Namespace) ListByLabel(labels string) (*corev1.NamespaceList, error) {
	listOptions := n.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return n.clientset.CoreV1().Namespaces().List(n.ctx, *listOptions)
}

// List list namespaces by labels, alias to "ListByLabel"
func (n *Namespace) List(labels string) (*corev1.NamespaceList, error) {
	return n.ListByLabel(labels)
}

// ListAll list all namespaces in the k8s cluster
func (n *Namespace) ListAll(labels string) (*corev1.NamespaceList, error) {
	return n.ListByLabel("")
}

// WatchByName watch namespace by name
func (n *Namespace) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = n.clientset.CoreV1().Namespaces().Watch(n.ctx, listOptions); err != nil {
			return
		}
		if _, err = n.Get(name); err != nil {
			isExist = false // namespace not exist bool
		} else {
			isExist = true // namespace exist
		}
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				if !isExist {
					addFunc(x)
				}
				isExist = true
			case watch.Modified:
				modifyFunc(x)
				isExist = true
			case watch.Deleted:
				deleteFunc(x)
				isExist = false
			case watch.Bookmark:
				log.Debug("watch namespace: bookmark.")
			case watch.Error:
				log.Debug("watch namespace: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch namespace: reconnect to kubernetes")
	}
}

// WatchByLabel watch namespace by labelSelector
func (n *Namespace) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher       watch.Interface
		namespaceList *corev1.NamespaceList
		timeout       = int64(0)
		isExist       bool
	)
	for {
		if watcher, err = n.clientset.CoreV1().Namespaces().Watch(n.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if namespaceList, err = n.List(labelSelector); err != nil {
			return
		}
		if len(namespaceList.Items) == 0 {
			isExist = false // namespace not exist
		} else {
			isExist = true // namespace exist
		}
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				if !isExist {
					addFunc(x)
				}
				isExist = true
			case watch.Modified:
				modifyFunc(x)
				isExist = true
			case watch.Deleted:
				deleteFunc(x)
				isExist = false
			case watch.Bookmark:
				log.Debug("watch namespace: bookmark.")
			case watch.Error:
				log.Debug("watch namespace: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch namespace: reconnect to kubernetes")
	}
}

// Watch watch namespace by name, alias to "WatchByName"
func (n *Namespace) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	n.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
	return n.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
