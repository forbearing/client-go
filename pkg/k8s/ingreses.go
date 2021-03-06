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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type Ingress struct {
	kubeconfig string
	namespace  string

	ctx             context.Context
	config          *rest.Config
	restClient      *rest.RESTClient
	clientset       *kubernetes.Clientset
	dynamicClient   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient
	informerFactory informers.SharedInformerFactory
	informer        cache.SharedIndexInformer
	Options         *HandlerOptions

	sync.Mutex
}

// new a ingress handler from kubeconfig or in-cluster config
func NewIngress(ctx context.Context, namespace, kubeconfig string) (ingress *Ingress, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	ingress = &Ingress{}

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
	ingress.kubeconfig = kubeconfig
	ingress.namespace = namespace
	ingress.ctx = ctx
	ingress.config = config
	ingress.restClient = restClient
	ingress.clientset = clientset
	ingress.dynamicClient = dynamicClient
	ingress.discoveryClient = discoveryClient
	ingress.informerFactory = informerFactory
	ingress.informer = informerFactory.Networking().V1().Ingresses().Informer()
	ingress.Options = &HandlerOptions{}

	return
}
func (i *Ingress) Namespace() string {
	return i.namespace
}
func (in *Ingress) DeepCopy() *Ingress {
	out := new(Ingress)

	out.kubeconfig = in.kubeconfig
	out.namespace = in.namespace

	out.ctx = in.ctx
	out.config = in.config
	out.restClient = in.restClient
	out.clientset = in.clientset
	out.dynamicClient = in.dynamicClient
	out.discoveryClient = in.discoveryClient
	out.informerFactory = in.informerFactory
	out.informer = in.informer

	out.Options = &HandlerOptions{}
	out.Options.ListOptions = *in.Options.ListOptions.DeepCopy()
	out.Options.GetOptions = *in.Options.GetOptions.DeepCopy()
	out.Options.CreateOptions = *in.Options.CreateOptions.DeepCopy()
	out.Options.UpdateOptions = *in.Options.UpdateOptions.DeepCopy()
	out.Options.PatchOptions = *in.Options.PatchOptions.DeepCopy()
	out.Options.ApplyOptions = *in.Options.ApplyOptions.DeepCopy()

	return out
}
func (i *Ingress) setNamespace(namespace string) {
	i.Lock()
	defer i.Unlock()
	i.namespace = namespace
}
func (i *Ingress) WithNamespace(namespace string) *Ingress {
	ingress := i.DeepCopy()
	ingress.setNamespace(namespace)
	return ingress
}
func (i *Ingress) WithDryRun() *Ingress {
	ingress := i.DeepCopy()
	ingress.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	ingress.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	ingress.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	ingress.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	ingress.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return ingress
}
func (i *Ingress) SetTimeout(timeout int64) {
	i.Lock()
	defer i.Unlock()
	i.Options.ListOptions.TimeoutSeconds = &timeout
}
func (i *Ingress) SetLimit(limit int64) {
	i.Lock()
	defer i.Unlock()
	i.Options.ListOptions.Limit = limit
}
func (i *Ingress) SetForceDelete(force bool) {
	i.Lock()
	defer i.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		i.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		i.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create ingress from map[string]interface{}
func (i *Ingress) CreateFromRaw(raw map[string]interface{}) (*networkingv1.Ingress, error) {
	ingress := &networkingv1.Ingress{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, ingress)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(ingress.Namespace) != 0 {
		namespace = ingress.Namespace
	} else {
		namespace = i.namespace
	}

	return i.clientset.NetworkingV1().Ingresses(namespace).Create(i.ctx, ingress, i.Options.CreateOptions)
}

// CreateFromBytes create ingress from bytes
func (i *Ingress) CreateFromBytes(data []byte) (*networkingv1.Ingress, error) {
	ingressJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	ingress := &networkingv1.Ingress{}
	err = json.Unmarshal(ingressJson, ingress)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(ingress.Namespace) != 0 {
		namespace = ingress.Namespace
	} else {
		namespace = i.namespace
	}

	return i.clientset.NetworkingV1().Ingresses(namespace).Create(i.ctx, ingress, i.Options.CreateOptions)
}

// CreateFromFile create ingress from yaml file
func (i *Ingress) CreateFromFile(path string) (*networkingv1.Ingress, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return i.CreateFromBytes(data)
}

// Create create ingress from file, alias to "CreateFromFile"
func (i *Ingress) Create(path string) (*networkingv1.Ingress, error) {
	return i.CreateFromFile(path)
}

// UpdateFromRaw update ingress from map[string]interface{}
func (i *Ingress) UpdateFromRaw(raw map[string]interface{}) (*networkingv1.Ingress, error) {
	ingress := &networkingv1.Ingress{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, ingress)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(ingress.Namespace) != 0 {
		namespace = ingress.Namespace
	} else {
		namespace = i.namespace
	}

	return i.clientset.NetworkingV1().Ingresses(namespace).Update(i.ctx, ingress, i.Options.UpdateOptions)
}

// UpdateFromBytes update ingress from bytes
func (i *Ingress) UpdateFromBytes(data []byte) (*networkingv1.Ingress, error) {
	ingressJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	ingress := &networkingv1.Ingress{}
	if err = json.Unmarshal(ingressJson, ingress); err != nil {
		return nil, err
	}

	var namespace string
	if len(ingress.Namespace) != 0 {
		namespace = ingress.Namespace
	} else {
		namespace = i.namespace
	}

	return i.clientset.NetworkingV1().Ingresses(namespace).Update(i.ctx, ingress, i.Options.UpdateOptions)
}

// UpdateFromFile update ingress from yaml file
func (i *Ingress) UpdateFromFile(path string) (*networkingv1.Ingress, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return i.UpdateFromBytes(data)
}

// Update update ingress from file, alias to "UpdateFromFile"
func (i *Ingress) Update(path string) (*networkingv1.Ingress, error) {
	return i.UpdateFromFile(path)
}

// ApplyFromRaw apply ingress from map[string]interface{}
func (i *Ingress) ApplyFromRaw(raw map[string]interface{}) (*networkingv1.Ingress, error) {
	ingress := &networkingv1.Ingress{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, ingress)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(ingress.Namespace) != 0 {
		namespace = ingress.Namespace
	} else {
		namespace = i.namespace
	}

	ingress, err = i.clientset.NetworkingV1().Ingresses(namespace).Create(i.ctx, ingress, i.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		ingress, err = i.clientset.NetworkingV1().Ingresses(namespace).Update(i.ctx, ingress, i.Options.UpdateOptions)
	}
	return ingress, err
}

// ApplyFromBytes apply ingress from bytes
func (i *Ingress) ApplyFromBytes(data []byte) (ingress *networkingv1.Ingress, err error) {
	ingress, err = i.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		ingress, err = i.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply ingress from yaml file
func (i *Ingress) ApplyFromFile(path string) (ingress *networkingv1.Ingress, err error) {
	ingress, err = i.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		ingress, err = i.UpdateFromFile(path)
	}
	return
}

// Apply apply ingress from file, alias to "ApplyFromFile"
func (i *Ingress) Apply(path string) (*networkingv1.Ingress, error) {
	return i.ApplyFromFile(path)
}

// DeleteFromBytes delete ingress from bytes
func (i *Ingress) DeleteFromBytes(data []byte) error {
	ingressJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	ingress := &networkingv1.Ingress{}
	err = json.Unmarshal(ingressJson, ingress)
	if err != nil {
		return err
	}

	var namespace string
	if len(ingress.Namespace) != 0 {
		namespace = ingress.Namespace
	} else {
		namespace = i.namespace
	}

	return i.WithNamespace(namespace).DeleteByName(ingress.Name)
}

// DeleteFromFile delete ingress from yaml file
func (i *Ingress) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return i.DeleteFromBytes(data)
}

// DeleteByName delete ingress by name
func (i *Ingress) DeleteByName(name string) error {
	return i.clientset.NetworkingV1().Ingresses(i.namespace).Delete(i.ctx, name, i.Options.DeleteOptions)
}

// Delete delete ingress by name, alias to "DeleteByName"
func (i *Ingress) Delete(name string) error {
	return i.DeleteByName(name)
}

// GetFromBytes get ingress from bytes
func (i *Ingress) GetFromBytes(data []byte) (*networkingv1.Ingress, error) {
	ingressJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}
	ingress := &networkingv1.Ingress{}
	if err = json.Unmarshal(ingressJson, ingress); err != nil {
		return nil, err
	}

	var namespace string
	if len(ingress.Namespace) != 0 {
		namespace = ingress.Namespace
	} else {
		namespace = i.namespace
	}
	return i.WithNamespace(namespace).GetByName(ingress.Name)
}

// GetFromFile get ingress from yaml file
func (i *Ingress) GetFromFile(path string) (*networkingv1.Ingress, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return i.GetFromBytes(data)
}

// GetByName get ingress by name
func (i *Ingress) GetByName(name string) (*networkingv1.Ingress, error) {
	return i.clientset.NetworkingV1().Ingresses(i.namespace).Get(i.ctx, name, i.Options.GetOptions)
}

// Get get ingress by name, alias to "GetByName"
func (i *Ingress) Get(name string) (*networkingv1.Ingress, error) {
	return i.GetByName(name)
}

// ListByLabel list ingresses by labels
func (i *Ingress) ListByLabel(labels string) (*networkingv1.IngressList, error) {
	listOptions := i.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return i.clientset.NetworkingV1().Ingresses(i.namespace).List(i.ctx, *listOptions)
}

// List list ingresses by labels, alias to "ListByLabel"
func (i *Ingress) List(labels string) (*networkingv1.IngressList, error) {
	return i.ListByLabel(labels)
}

// ListByNamespace list ingresses by namespace
func (i *Ingress) ListByNamespace(namespace string) (*networkingv1.IngressList, error) {
	return i.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all ingresses in the k8s cluster
func (i *Ingress) ListAll() (*networkingv1.IngressList, error) {
	return i.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// WatchByName watch ingress by name
func (i *Ingress) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: i.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = i.clientset.NetworkingV1().Ingresses(i.namespace).Watch(i.ctx, listOptions); err != nil {
			return
		}
		if _, err = i.Get(name); err != nil {
			isExist = false // configmap not exist
		} else {
			isExist = true // configmap exist
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
				log.Debug("watch ingress: bookmark.")
			case watch.Error:
				log.Debug("watch ingress: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch ingress: reconnect to kubernetes")
	}
}

// WatchByLabel watch ingress by labelSelector
func (i *Ingress) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher     watch.Interface
		ingressList *networkingv1.IngressList
		timeout     = int64(0)
		isExist     bool
	)
	for {
		if watcher, err = i.clientset.NetworkingV1().Ingresses(i.namespace).Watch(i.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if ingressList, err = i.List(labelSelector); err != nil {
			return
		}
		if len(ingressList.Items) == 0 {
			isExist = false // ingress not exist
		} else {
			isExist = true // ingress exist
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
				log.Debug("watch ingress: bookmark.")
			case watch.Error:
				log.Debug("watch ingress: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch ingress: reconnect to kubernetes")
	}
}

// Watch watch ingress by name, alias to "WatchByName"
func (i *Ingress) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return i.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}

// RunInformer
func (i *Ingress) RunInformer(
	addFunc func(obj interface{}),
	updateFunc func(oldObj, newObj interface{}),
	deleteFunc func(obj interface{}),
	stopCh chan struct{}) {
	i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    addFunc,
		UpdateFunc: updateFunc,
		DeleteFunc: deleteFunc,
	})
	i.informer.Run(stopCh)
}
