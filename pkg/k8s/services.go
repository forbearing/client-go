package k8s

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
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

type Service struct {
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

// new a Service handler from kubeconfig or in-cluster config
func NewService(ctx context.Context, namespace, kubeconfig string) (service *Service, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	service = &Service{}

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

	if len(namespace) == 0 {
		namespace = metav1.NamespaceDefault
	}
	service.kubeconfig = kubeconfig
	service.namespace = namespace
	service.ctx = ctx
	service.config = config
	service.restClient = restClient
	service.clientset = clientset
	service.dynamicClient = dynamicClient
	service.discoveryClient = discoveryClient
	service.informerFactory = informerFactory
	service.Options = &HandlerOptions{}

	return
}
func (s *Service) Namespace() string {
	return s.namespace
}
func (in *Service) DeepCopy() *Service {
	out := new(Service)

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
func (s *Service) setNamespace(namespace string) {
	s.Lock()
	defer s.Unlock()
	s.namespace = namespace
}
func (s *Service) WithNamespace(namespace string) *Service {
	service := s.DeepCopy()
	service.setNamespace(namespace)
	return service
}
func (s *Service) WithDryRun() *Service {
	svc := s.DeepCopy()
	svc.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	svc.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	svc.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	svc.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	svc.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return svc
}
func (s *Service) SetTimeout(timeout int64) {
	s.Lock()
	defer s.Unlock()
	s.Options.ListOptions.TimeoutSeconds = &timeout
}
func (s *Service) SetLimit(limit int64) {
	s.Lock()
	defer s.Unlock()
	s.Options.ListOptions.Limit = limit
}
func (s *Service) SetForceDelete(force bool) {
	s.Lock()
	defer s.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		s.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		s.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create service from map[string]interface{}
func (s *Service) CreateFromRaw(raw map[string]interface{}) (*corev1.Service, error) {
	service := &corev1.Service{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, service)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(service.Namespace) != 0 {
		namespace = service.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.CoreV1().Services(namespace).Create(s.ctx, service, s.Options.CreateOptions)
}

// CreateFromBytes create service from bytes
func (s *Service) CreateFromBytes(data []byte) (*corev1.Service, error) {
	serviceJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	service := &corev1.Service{}
	err = json.Unmarshal(serviceJson, service)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(service.Namespace) != 0 {
		namespace = service.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.CoreV1().Services(namespace).Create(s.ctx, service, s.Options.CreateOptions)
}

// CreateFromFile create service from yaml file
func (s *Service) CreateFromFile(path string) (*corev1.Service, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.CreateFromBytes(data)
}

// Create create service from yaml file, alias to "CreateFromFile"
func (s *Service) Create(path string) (*corev1.Service, error) {
	return s.CreateFromFile(path)
}

// UpdateFromRaw update service from map[string]interface{}
func (s *Service) UpdateFromRaw(raw map[string]interface{}) (*corev1.Service, error) {
	service := &corev1.Service{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, service)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(service.Namespace) != 0 {
		namespace = service.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.CoreV1().Services(namespace).Update(s.ctx, service, s.Options.UpdateOptions)
}

// UpdateFromBytes update service from bytes
func (s *Service) UpdateFromBytes(data []byte) (*corev1.Service, error) {
	serviceJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	service := &corev1.Service{}
	err = json.Unmarshal(serviceJson, service)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(service.Namespace) != 0 {
		namespace = service.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.CoreV1().Services(namespace).Update(s.ctx, service, s.Options.UpdateOptions)
}

// UpdateFromFile update service from yaml file
func (s *Service) UpdateFromFile(path string) (*corev1.Service, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.UpdateFromBytes(data)
}

// Update update service from yaml file, alias to "UpdateFromFile"
func (s *Service) Update(path string) (*corev1.Service, error) {
	return s.UpdateFromFile(path)
}

// ApplyFromRaw apply service from map[string]interface{}
func (s *Service) ApplyFromRaw(raw map[string]interface{}) (*corev1.Service, error) {
	service := &corev1.Service{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, service)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(service.Namespace) != 0 {
		namespace = service.Namespace
	} else {
		namespace = s.namespace
	}

	service, err = s.clientset.CoreV1().Services(namespace).Create(s.ctx, service, s.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		service, err = s.clientset.CoreV1().Services(namespace).Update(s.ctx, service, s.Options.UpdateOptions)
	}
	return service, err
}

// ApplyFromBytes apply service from bytes
func (s *Service) ApplyFromBytes(data []byte) (service *corev1.Service, err error) {
	service, err = s.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		service, err = s.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply service from yaml file
func (s *Service) ApplyFromFile(path string) (service *corev1.Service, err error) {
	service, err = s.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		service, err = s.UpdateFromFile(path)
	}
	return
}

// Apply apply service from yaml file, alias to "ApplyFromFile"
func (s *Service) Apply(path string) (*corev1.Service, error) {
	return s.ApplyFromFile(path)
}

// DeleteFromBytes delete service from bytes
func (s *Service) DeleteFromBytes(data []byte) error {
	serviceJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	service := &corev1.Service{}
	err = json.Unmarshal(serviceJson, service)
	if err != nil {
		return err
	}

	var namespace string
	if len(service.Namespace) != 0 {
		namespace = service.Namespace
	} else {
		namespace = s.namespace
	}

	return s.WithNamespace(namespace).DeleteByName(service.Name)
}

// DeleteFromFile delete service from yaml file
func (s *Service) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return s.DeleteFromBytes(data)
}

// DeleteByName delete service by name
func (s *Service) DeleteByName(name string) error {
	return s.clientset.CoreV1().Services(s.namespace).Delete(s.ctx, name, s.Options.DeleteOptions)
}

// Delete delete service by name, alias to "DeleteByName"
func (s *Service) Delete(name string) error {
	return s.DeleteByName(name)
}

// GetFromBytes get service from bytes
func (s *Service) GetFromBytes(data []byte) (*corev1.Service, error) {
	serviceJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	service := &corev1.Service{}
	err = json.Unmarshal(serviceJson, service)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(service.Namespace) != 0 {
		namespace = service.Namespace
	} else {
		namespace = s.namespace
	}

	return s.WithNamespace(namespace).GetByName(service.Name)
}

// GetFromFile get service from yaml file
func (s *Service) GetFromFile(path string) (*corev1.Service, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.GetFromBytes(data)
}

// GetByName get service by name
func (s *Service) GetByName(name string) (*corev1.Service, error) {
	return s.clientset.CoreV1().Services(s.namespace).Get(s.ctx, name, s.Options.GetOptions)
}

// Get get service by name, alias to "GetByName"
func (s *Service) Get(name string) (*corev1.Service, error) {
	return s.GetByName(name)
}

// ListByLabel list services by labels
func (s *Service) ListByLabel(labels string) (*corev1.ServiceList, error) {
	listOptions := s.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return s.clientset.CoreV1().Services(s.namespace).List(s.ctx, *listOptions)
}

// List list services by labels, alias to "ListByLabel"
func (s *Service) List(labels string) (*corev1.ServiceList, error) {
	return s.ListByLabel(labels)
}

// ListByNamespace list services by labels, alias to "ListByLabel"
func (s *Service) ListByNamespace(namespace string) (*corev1.ServiceList, error) {
	return s.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all services in the k8s cluster
func (s *Service) ListAll() (*corev1.ServiceList, error) {
	return s.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// WatchByName watch services by name
func (s *Service) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: s.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = s.clientset.CoreV1().Services(s.namespace).Watch(s.ctx, listOptions); err != nil {
			logrus.Error(err)
			return
		}
		if _, err = s.Get(name); err != nil {
			isExist = false // service not exist
		} else {
			isExist = true // service exist
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
				log.Debug("watch service: bookmark.")
			case watch.Error:
				log.Debug("watch service: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch service: reconnect to kubernetes")
	}
}

// WatchByLabel watch services by labelSelector
func (s *Service) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher     watch.Interface
		serviceList *corev1.ServiceList
		timeout     = int64(0)
		isExist     bool
	)
	for {
		if watcher, err = s.clientset.CoreV1().Services(s.namespace).Watch(s.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			logrus.Error(err)
			return
		}
		if serviceList, err = s.List(labelSelector); err != nil {
			logrus.Error(err)
			return
		}
		if len(serviceList.Items) == 0 {
			isExist = false // service not exist
		} else {
			isExist = true // service exist
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
				log.Debug("watch service: bookmark.")
			case watch.Error:
				log.Debug("watch service: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch service: reconnect to kubernetes")
	}
}

// Watch watch services by name, alias to "WatchByName"
func (s *Service) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return s.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
