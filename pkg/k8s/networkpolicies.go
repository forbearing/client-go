package k8s

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	networkingv1 "k8s.io/api/networking/v1"
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

type NetworkPolicy struct {
	kubeconfig string
	namespace  string

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

// new a networkpolicy handler from kubeconfig or in-cluster config
func NewNetworkPolicy(ctx context.Context, namespace, kubeconfig string) (netpol *NetworkPolicy, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	netpol = &NetworkPolicy{}

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
	config.GroupVersion = &networkingv1.SchemeGroupVersion
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

	if len(namespace) == 0 {
		namespace = metav1.NamespaceDefault
	}
	netpol.kubeconfig = kubeconfig
	netpol.namespace = namespace
	netpol.ctx = ctx
	netpol.config = config
	netpol.restClient = restClient
	netpol.clientset = clientset
	netpol.dynamicClient = dynamicClient
	netpol.discoveryClient = discoveryClient
	netpol.informerFactory = informerFactory
	netpol.Options = &HandlerOptions{}

	return
}
func (n *NetworkPolicy) Namespace() string {
	return n.namespace
}
func (in *NetworkPolicy) DeepCopy() *NetworkPolicy {
	out := new(NetworkPolicy)

	out.kubeconfig = in.kubeconfig
	out.namespace = in.namespace

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
func (n *NetworkPolicy) setNamespace(namespace string) {
	n.Lock()
	defer n.Unlock()
	n.namespace = namespace
}
func (n *NetworkPolicy) WithNamespace(namespace string) *NetworkPolicy {
	netpol := n.DeepCopy()
	netpol.setNamespace(namespace)
	return netpol
}
func (n *NetworkPolicy) WithDryRun() *NetworkPolicy {
	netpol := n.DeepCopy()
	netpol.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	netpol.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	netpol.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	netpol.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	netpol.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return netpol
}
func (n *NetworkPolicy) SetTimeout(timeout int64) {
	n.Lock()
	defer n.Unlock()
	n.Options.ListOptions.TimeoutSeconds = &timeout
}
func (n *NetworkPolicy) SetLimit(limit int64) {
	n.Lock()
	defer n.Unlock()
	n.Options.ListOptions.Limit = limit
}
func (n *NetworkPolicy) SetForceDelete(force bool) {
	n.Lock()
	defer n.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		n.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		n.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create networkpolicy from map[string]interface{}
func (n *NetworkPolicy) CreateFromRaw(raw map[string]interface{}) (*networkingv1.NetworkPolicy, error) {
	netpol := &networkingv1.NetworkPolicy{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, netpol)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(netpol.Namespace) != 0 {
		namespace = netpol.Namespace
	} else {
		namespace = n.namespace
	}

	return n.clientset.NetworkingV1().NetworkPolicies(namespace).Create(n.ctx, netpol, n.Options.CreateOptions)
}

// CreateFromBytes create networkpolicy from bytes
func (n *NetworkPolicy) CreateFromBytes(data []byte) (*networkingv1.NetworkPolicy, error) {
	netpolJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	netpol := &networkingv1.NetworkPolicy{}
	err = json.Unmarshal(netpolJson, netpol)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(netpol.Namespace) != 0 {
		namespace = netpol.Namespace
	} else {
		namespace = n.namespace
	}

	return n.clientset.NetworkingV1().NetworkPolicies(namespace).Create(n.ctx, netpol, n.Options.CreateOptions)
}

// CreateFromFile create networkpolicy from yaml file
func (n *NetworkPolicy) CreateFromFile(path string) (*networkingv1.NetworkPolicy, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return n.CreateFromBytes(data)
}

// Create create networkpolicy from yaml file, alias to "CreateFromFile"
func (n *NetworkPolicy) Create(path string) (*networkingv1.NetworkPolicy, error) {
	return n.CreateFromFile(path)
}

// UpdateFromRaw update networkpolicy from map[string]interface{}
func (n *NetworkPolicy) UpdateFromRaw(raw map[string]interface{}) (*networkingv1.NetworkPolicy, error) {
	netpol := &networkingv1.NetworkPolicy{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, netpol)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(netpol.Namespace) != 0 {
		namespace = netpol.Namespace
	} else {
		namespace = n.namespace
	}

	return n.clientset.NetworkingV1().NetworkPolicies(namespace).Update(n.ctx, netpol, n.Options.UpdateOptions)
}

// UpdateFromBytes update networkpolicy from bytes
func (n *NetworkPolicy) UpdateFromBytes(data []byte) (*networkingv1.NetworkPolicy, error) {
	netpolJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	netpol := &networkingv1.NetworkPolicy{}
	if err = json.Unmarshal(netpolJson, netpol); err != nil {
		return nil, err
	}

	var namespace string
	if len(netpol.Namespace) != 0 {
		namespace = netpol.Namespace
	} else {
		namespace = n.namespace
	}
	return n.clientset.NetworkingV1().NetworkPolicies(namespace).Update(n.ctx, netpol, n.Options.UpdateOptions)
}

// UpdateFromFile update networkpolicy from yaml file
func (n *NetworkPolicy) UpdateFromFile(path string) (*networkingv1.NetworkPolicy, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return n.UpdateFromBytes(data)
}

// Update update networkpolicy from yaml file, alias to "UpdateFromFile"
func (n *NetworkPolicy) Update(path string) (*networkingv1.NetworkPolicy, error) {
	return n.UpdateFromFile(path)
}

// ApplyFromRaw apply netpol from map[string]interface{}
func (n *NetworkPolicy) ApplyFromRaw(raw map[string]interface{}) (*networkingv1.NetworkPolicy, error) {
	netpol := &networkingv1.NetworkPolicy{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, netpol)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(netpol.Namespace) != 0 {
		namespace = netpol.Namespace
	} else {
		namespace = n.namespace
	}

	netpol, err = n.clientset.NetworkingV1().NetworkPolicies(namespace).Create(n.ctx, netpol, n.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		netpol, err = n.clientset.NetworkingV1().NetworkPolicies(namespace).Update(n.ctx, netpol, n.Options.UpdateOptions)
	}
	return netpol, err
}

// ApplyFromBytes apply networkpolicy from bytes
func (n *NetworkPolicy) ApplyFromBytes(data []byte) (netpol *networkingv1.NetworkPolicy, err error) {
	netpol, err = n.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		netpol, err = n.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply netpol from yaml file
func (n *NetworkPolicy) ApplyFromFile(path string) (netpol *networkingv1.NetworkPolicy, err error) {
	netpol, err = n.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		netpol, err = n.UpdateFromFile(path)
	}
	return
}

// Apply apply networkpolicy from yaml file, alias to "ApplyFromFile"
func (n *NetworkPolicy) Apply(path string) (*networkingv1.NetworkPolicy, error) {
	return n.ApplyFromFile(path)
}

// DeleteFromBytes delete networkpolicy from bytes
func (n *NetworkPolicy) DeleteFromBytes(data []byte) error {
	netpolJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	netpol := &networkingv1.NetworkPolicy{}
	err = json.Unmarshal(netpolJson, netpol)
	if err != nil {
		return err
	}

	var namespace string
	if len(netpol.Namespace) != 0 {
		namespace = netpol.Namespace
	} else {
		namespace = n.namespace
	}

	return n.WithNamespace(namespace).DeleteByName(netpol.Name)
}

// DeleteFromFile delete networkpolicy from yaml file
func (n *NetworkPolicy) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return n.DeleteFromBytes(data)
}

// DeleteByName delete networkpolicy by name
func (n *NetworkPolicy) DeleteByName(name string) error {
	return n.clientset.NetworkingV1().NetworkPolicies(n.namespace).Delete(n.ctx, name, n.Options.DeleteOptions)
}

// Delete delete networkpolicy by name, alias to "DeleteByName"
func (n *NetworkPolicy) Delete(name string) error {
	return n.DeleteByName(name)
}

// GetFromBytes get networkpolicy from bytes
func (n *NetworkPolicy) GetFromBytes(data []byte) (*networkingv1.NetworkPolicy, error) {
	netpolJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	netpol := &networkingv1.NetworkPolicy{}
	err = json.Unmarshal(netpolJson, netpol)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(netpol.Namespace) != 0 {
		namespace = netpol.Namespace
	} else {
		namespace = n.namespace
	}

	return n.WithNamespace(namespace).GetByName(netpol.Name)
}

// GetFromBytes get networkpolicy from yaml file
func (n *NetworkPolicy) GetFromFile(path string) (*networkingv1.NetworkPolicy, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return n.GetFromBytes(data)
}

// GetByName get networkpolicy by name
func (n *NetworkPolicy) GetByName(name string) (*networkingv1.NetworkPolicy, error) {
	return n.clientset.NetworkingV1().NetworkPolicies(n.namespace).Get(n.ctx, name, n.Options.GetOptions)
}

// Get get networkpolicy by name, alias to "GetByName"
func (n *NetworkPolicy) Get(name string) (*networkingv1.NetworkPolicy, error) {
	return n.GetByName(name)
}

// ListByLabel list networkpolicies by labels
func (n *NetworkPolicy) ListByLabel(labels string) (*networkingv1.NetworkPolicyList, error) {
	listOptions := n.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return n.clientset.NetworkingV1().NetworkPolicies(n.namespace).List(n.ctx, *listOptions)
}

// List list networkpolicies by labels, alias to "ListByLabel"
func (n *NetworkPolicy) List(labels string) (*networkingv1.NetworkPolicyList, error) {
	return n.ListByLabel(labels)
}

// ListByNamespace list networkpolicies by namespace
func (n *NetworkPolicy) ListByNamespace(namespace string) (*networkingv1.NetworkPolicyList, error) {
	return n.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all networkpolicies in the k8s cluster
func (n *NetworkPolicy) ListAll() (*networkingv1.NetworkPolicyList, error) {
	return n.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// WatchByName watch networkpolicyies by name
func (n *NetworkPolicy) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: n.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = n.clientset.NetworkingV1().NetworkPolicies(n.namespace).Watch(n.ctx, listOptions); err != nil {
			return
		}
		if _, err = n.Get(name); err != nil {
			isExist = false // networkpolicy not exist
		} else {
			isExist = true // networkpolicy exist
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
				log.Debug("watch networkpolicy: bookmark.")
			case watch.Error:
				log.Debug("watch networkpolicy: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch networkpolicy: reconnect to kubernetes")
	}
}

// WatchByLabel watch networkpolicies by labelSelector
func (n *NetworkPolicy) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher           watch.Interface
		networkpolicyList *networkingv1.NetworkPolicyList
		timeout           = int64(0)
		isExist           bool
	)
	for {
		if watcher, err = n.clientset.NetworkingV1().NetworkPolicies(n.namespace).Watch(n.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if networkpolicyList, err = n.List(labelSelector); err != nil {
			return
		}
		if len(networkpolicyList.Items) == 0 {
			isExist = false // networkpolicy not exist
		} else {
			isExist = true // networkpolicy exist
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
				log.Debug("watch networkpolicy: bookmark.")
			case watch.Error:
				log.Debug("watch networkpolicy: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch networkpolicy: reconnect to kubernetes")
	}
}

// Watch watch networkpolicies by name, alias to "WatchByName"
func (n *NetworkPolicy) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return n.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
