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

type IngressClass struct {
	kubeconfig string

	ctx             context.Context
	config          *rest.Config
	restClient      *rest.RESTClient
	clientset       *kubernetes.Clientset
	dynamicClient   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient
	informerFactory informers.SharedInformerFactory
	informer        cache.SharedIndexInformer

	Options *HandlerOptions

	sync.Mutex
}

// new a ingressclass handler from kubeconfig or in-cluster config
func NewIngressClass(ctx context.Context, kubeconfig string) (ingc *IngressClass, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	ingc = &IngressClass{}

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

	ingc.kubeconfig = kubeconfig
	ingc.ctx = ctx
	ingc.config = config
	ingc.restClient = restClient
	ingc.clientset = clientset
	ingc.dynamicClient = dynamicClient
	ingc.discoveryClient = discoveryClient
	ingc.informerFactory = informerFactory
	ingc.informer = informerFactory.Networking().V1().IngressClasses().Informer()
	ingc.Options = &HandlerOptions{}

	return
}
func (in *IngressClass) DeepCopy() *IngressClass {
	out := new(IngressClass)

	out.kubeconfig = in.kubeconfig

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
func (i *IngressClass) WithDryRun() *IngressClass {
	ingc := i.DeepCopy()
	ingc.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	ingc.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	ingc.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	ingc.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	ingc.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return ingc
}
func (i *IngressClass) SetTimeout(timeout int64) {
	i.Lock()
	defer i.Unlock()
	i.Options.ListOptions.TimeoutSeconds = &timeout
}
func (i *IngressClass) SetLimit(limit int64) {
	i.Lock()
	defer i.Unlock()
	i.Options.ListOptions.Limit = limit
}
func (i *IngressClass) SetForceDelete(force bool) {
	i.Lock()
	defer i.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		i.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		i.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create ingressclass from map[string]interface{}
func (i *IngressClass) CreateFromRaw(raw map[string]interface{}) (*networkingv1.IngressClass, error) {
	ingressclass := &networkingv1.IngressClass{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, ingressclass)
	if err != nil {
		return nil, err
	}

	return i.clientset.NetworkingV1().IngressClasses().Create(i.ctx, ingressclass, i.Options.CreateOptions)
}

// CreateFromBytes create ingressclass from bytes
func (i *IngressClass) CreateFromBytes(data []byte) (*networkingv1.IngressClass, error) {
	ingcJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	ingc := &networkingv1.IngressClass{}
	err = json.Unmarshal(ingcJson, ingc)
	if err != nil {
		return nil, err
	}

	return i.clientset.NetworkingV1().IngressClasses().Create(i.ctx, ingc, i.Options.CreateOptions)
}

// CreateFromFile create ingressclass from yaml file
func (i *IngressClass) CreateFromFile(path string) (*networkingv1.IngressClass, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return i.CreateFromBytes(data)
}

// Create create ingressclass from yaml file, alias to "CreateFromFile"
func (i *IngressClass) Create(path string) (*networkingv1.IngressClass, error) {
	return i.CreateFromFile(path)
}

// UpdateFromRaw update ingressclass from map[string]interface{}
func (i *IngressClass) UpdateFromRaw(raw map[string]interface{}) (*networkingv1.IngressClass, error) {
	ingressclass := &networkingv1.IngressClass{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, ingressclass)
	if err != nil {
		return nil, err
	}

	return i.clientset.NetworkingV1().IngressClasses().Update(i.ctx, ingressclass, i.Options.UpdateOptions)
}

// UpdateFromBytes update ingressclass from bytes
func (i *IngressClass) UpdateFromBytes(data []byte) (*networkingv1.IngressClass, error) {
	ingcJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	ingc := &networkingv1.IngressClass{}
	err = json.Unmarshal(ingcJson, ingc)
	if err != nil {
		return nil, err
	}

	return i.clientset.NetworkingV1().IngressClasses().Update(i.ctx, ingc, i.Options.UpdateOptions)
}

// UpdateFromFile update ingressclass from yaml file
func (i *IngressClass) UpdateFromFile(path string) (*networkingv1.IngressClass, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return i.UpdateFromBytes(data)
}

// Update update ingressclass from yaml file, alias to "UpdateFromFile"
func (i *IngressClass) Update(path string) (*networkingv1.IngressClass, error) {
	return i.UpdateFromFile(path)
}

// ApplyFromRaw apply ingressclass from map[string]interface{}
func (i *IngressClass) ApplyFromRaw(raw map[string]interface{}) (*networkingv1.IngressClass, error) {
	ingressclass := &networkingv1.IngressClass{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, ingressclass)
	if err != nil {
		return nil, err
	}

	ingressclass, err = i.clientset.NetworkingV1().IngressClasses().Create(i.ctx, ingressclass, i.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		ingressclass, err = i.clientset.NetworkingV1().IngressClasses().Update(i.ctx, ingressclass, i.Options.UpdateOptions)
	}
	return ingressclass, err
}

// ApplyFromBytes apply ingressclass from bytes
func (i *IngressClass) ApplyFromBytes(data []byte) (ingc *networkingv1.IngressClass, err error) {
	ingc, err = i.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		ingc, err = i.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply ingressclass from yaml file
func (i *IngressClass) ApplyFromFile(path string) (ingc *networkingv1.IngressClass, err error) {
	ingc, err = i.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		ingc, err = i.UpdateFromFile(path)
	}
	return
}

// Apply apply ingressclass from yaml file, alias to "ApplyFromFile"
func (i *IngressClass) Apply(path string) (*networkingv1.IngressClass, error) {
	return i.ApplyFromFile(path)
}

// DeleteFromBytes delete ingressclass from bytes
func (i *IngressClass) DeleteFromBytes(data []byte) error {
	ingcJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	ingc := &networkingv1.IngressClass{}
	err = json.Unmarshal(ingcJson, ingc)
	if err != nil {
		return err
	}

	return i.DeleteByName(ingc.Name)
}

// DeleteFromFile delete ingressclass from yaml file
func (i *IngressClass) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return i.DeleteFromBytes(data)
}

// DeleteByName delete ingressclass by name
func (i *IngressClass) DeleteByName(name string) error {
	return i.clientset.NetworkingV1().IngressClasses().Delete(i.ctx, name, i.Options.DeleteOptions)
}

// Delete delete ingressclass by name, alias to "DeleteByName"
func (i *IngressClass) Delete(name string) error {
	return i.DeleteByName(name)
}

// GetFromBytes get ingressclass from bytes
func (i *IngressClass) GetFromBytes(data []byte) (*networkingv1.IngressClass, error) {
	ingcJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	ingc := &networkingv1.IngressClass{}
	err = json.Unmarshal(ingcJson, ingc)
	if err != nil {
		return nil, err
	}

	return i.GetByName(ingc.Name)
}

// GetFromFile get ingressclass from yaml file
func (i *IngressClass) GetFromFile(path string) (*networkingv1.IngressClass, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return i.GetFromBytes(data)
}

// GetByName get ingressclass by name
func (i *IngressClass) GetByName(name string) (*networkingv1.IngressClass, error) {
	return i.clientset.NetworkingV1().IngressClasses().Get(i.ctx, name, i.Options.GetOptions)
}

// Get get ingressclass by name, alias to "GetByName"
func (i *IngressClass) Get(name string) (*networkingv1.IngressClass, error) {
	return i.clientset.NetworkingV1().IngressClasses().Get(i.ctx, name, i.Options.GetOptions)
}

// ListByLabel list ingressclasses by labels
func (i *IngressClass) ListByLabel(labels string) (*networkingv1.IngressClassList, error) {
	listOptions := i.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return i.clientset.NetworkingV1().IngressClasses().List(i.ctx, *listOptions)
}

// List list ingressclasses by labels, alias to "ListByLabel"
func (i *IngressClass) List(labels string) (*networkingv1.IngressClassList, error) {
	return i.ListByLabel(labels)
}

// ListAll list all ingressclasses in the k8s cluster
func (i *IngressClass) ListAll(labels string) (*networkingv1.IngressClassList, error) {
	return i.ListByLabel("")
}

// WatchByName watch ingressclass by name
func (i *IngressClass) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = i.clientset.NetworkingV1().IngressClasses().Watch(i.ctx, listOptions); err != nil {
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
				log.Debug("watch ingressclass: bookmark.")
			case watch.Error:
				log.Debug("watch ingressclass: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch ingressclass: reconnect to kubernetes")
	}
}

// WatchByLabel watch ingressclass by labelSelector
func (i *IngressClass) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher          watch.Interface
		ingressclassList *networkingv1.IngressClassList
		timeout          = int64(0)
		isExist          bool
	)
	for {
		if watcher, err = i.clientset.NetworkingV1().IngressClasses().Watch(i.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if ingressclassList, err = i.List(labelSelector); err != nil {
			return
		}
		if len(ingressclassList.Items) == 0 {
			isExist = false // ingressclass not exist
		} else {
			isExist = true // ingressclass exist
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
				log.Debug("watch ingressclass: bookmark.")
			case watch.Error:
				log.Debug("watch ingressclass: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch ingressclass: reconnect to kubernetes")
	}
}

// Watch watch ingressclass by name, alias to "WatchByName"
func (i *IngressClass) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return i.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}

// RunInformer
func (i *IngressClass) RunInformer(
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
