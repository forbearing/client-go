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

type ServiceAccount struct {
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

// new a ServiceAccount handler from kubeconfig or in-cluster config
func NewServiceAccount(ctx context.Context, namespace, kubeconfig string) (sa *ServiceAccount, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	sa = &ServiceAccount{}

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

	if len(namespace) == 0 {
		namespace = metav1.NamespaceDefault
	}
	sa.kubeconfig = kubeconfig
	sa.namespace = namespace
	sa.ctx = ctx
	sa.config = config
	sa.restClient = restClient
	sa.clientset = clientset
	sa.dynamicClient = dynamicClient
	sa.discoveryClient = discoveryClient
	sa.informerFactory = informerFactory
	sa.Options = &HandlerOptions{}

	return
}
func (s *ServiceAccount) Namespace() string {
	return s.namespace
}
func (in *ServiceAccount) DeepCopy() *ServiceAccount {
	out := new(ServiceAccount)

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
func (s *ServiceAccount) setNamespace(namespace string) {
	s.Lock()
	defer s.Unlock()
	s.namespace = namespace
}
func (s *ServiceAccount) WithNamespace(namespace string) *ServiceAccount {
	sa := s.DeepCopy()
	sa.setNamespace(namespace)
	return sa
}
func (s *ServiceAccount) WithDryRun() *ServiceAccount {
	sa := s.DeepCopy()
	sa.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	sa.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	sa.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	sa.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	sa.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return sa
}
func (s *ServiceAccount) SetTimeout(timeout int64) {
	s.Lock()
	defer s.Unlock()
	s.Options.ListOptions.TimeoutSeconds = &timeout
}
func (s *ServiceAccount) SetLimit(limit int64) {
	s.Lock()
	defer s.Unlock()
	s.Options.ListOptions.Limit = limit
}
func (s *ServiceAccount) SetForceDelete(force bool) {
	s.Lock()
	defer s.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		s.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		s.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create serviceaccount from map[string]interface{}
func (s *ServiceAccount) CreateFromRaw(raw map[string]interface{}) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, sa)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(sa.Namespace) != 0 {
		namespace = sa.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.CoreV1().ServiceAccounts(namespace).Create(s.ctx, sa, s.Options.CreateOptions)
}

// CreateFromBytes create serviceaccount from bytes
func (s *ServiceAccount) CreateFromBytes(data []byte) (*corev1.ServiceAccount, error) {
	saJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	sa := &corev1.ServiceAccount{}
	if err = json.Unmarshal(saJson, sa); err != nil {
		return nil, err
	}

	var namespace string
	if len(sa.Namespace) != 0 {
		namespace = sa.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.CoreV1().ServiceAccounts(namespace).Create(s.ctx, sa, s.Options.CreateOptions)
}

// CreateFromFile create serviceaccount from yaml file
func (s *ServiceAccount) CreateFromFile(path string) (*corev1.ServiceAccount, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.CreateFromBytes(data)
}

// Create create serviceaccount from yaml file, alias to "CreateFromFile"
func (s *ServiceAccount) Create(path string) (*corev1.ServiceAccount, error) {
	return s.CreateFromFile(path)
}

// UpdateFromRaw update serviceaccount from map[string]interface{}
func (s *ServiceAccount) UpdateFromRaw(raw map[string]interface{}) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, sa)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(sa.Namespace) != 0 {
		namespace = sa.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.CoreV1().ServiceAccounts(namespace).Update(s.ctx, sa, s.Options.UpdateOptions)
}

// UpdateFromBytes update serviceaccount from bytes
func (s *ServiceAccount) UpdateFromBytes(data []byte) (*corev1.ServiceAccount, error) {
	saJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	sa := &corev1.ServiceAccount{}
	err = json.Unmarshal(saJson, sa)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(sa.Namespace) != 0 {
		namespace = sa.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.CoreV1().ServiceAccounts(namespace).Update(s.ctx, sa, s.Options.UpdateOptions)
}

// UpdateFromFile update serviceaccount from yaml file
func (s *ServiceAccount) UpdateFromFile(path string) (*corev1.ServiceAccount, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.UpdateFromBytes(data)
}

// Update update serviceaccount from yaml file, alias to "UpdateFromFile"
func (s *ServiceAccount) Update(path string) (*corev1.ServiceAccount, error) {
	return s.UpdateFromFile(path)
}

// ApplyFromRaw apply serviceaccount from map[string]interface{}
func (s *ServiceAccount) ApplyFromRaw(raw map[string]interface{}) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, sa)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(sa.Namespace) != 0 {
		namespace = sa.Namespace
	} else {
		namespace = s.namespace
	}

	sa, err = s.clientset.CoreV1().ServiceAccounts(namespace).Create(s.ctx, sa, s.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		sa, err = s.clientset.CoreV1().ServiceAccounts(namespace).Update(s.ctx, sa, s.Options.UpdateOptions)
	}
	return sa, err
}

// ApplyFromBytes apply serviceaccount from file
func (s *ServiceAccount) ApplyFromBytes(data []byte) (serviceaccount *corev1.ServiceAccount, err error) {
	serviceaccount, err = s.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		serviceaccount, err = s.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply serviceaccount from yaml file
func (s *ServiceAccount) ApplyFromFile(path string) (serviceaccount *corev1.ServiceAccount, err error) {
	serviceaccount, err = s.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		serviceaccount, err = s.UpdateFromFile(path)
	}
	return
}

// Apply apply serviceaccount from yaml file, alias to "ApplyFromFile"
func (s *ServiceAccount) Apply(path string) (*corev1.ServiceAccount, error) {
	return s.ApplyFromFile(path)
}

// DeleteFromBytes delete serviceaccount from bytes
func (s *ServiceAccount) DeleteFromBytes(data []byte) error {
	saJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	sa := &corev1.ServiceAccount{}
	err = json.Unmarshal(saJson, sa)
	if err != nil {
		return err
	}

	var namespace string
	if len(sa.Namespace) != 0 {
		namespace = sa.Namespace
	} else {
		namespace = s.namespace
	}

	return s.WithNamespace(namespace).DeleteByName(sa.Name)
}

// DeleteFromFile delete serviceaccount from yaml file
func (s *ServiceAccount) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return s.DeleteFromBytes(data)
}

// DeleteByName delete serviceaccount by name
func (s *ServiceAccount) DeleteByName(name string) error {
	return s.clientset.CoreV1().ServiceAccounts(s.namespace).Delete(s.ctx, name, s.Options.DeleteOptions)
}

// Delete delete serviceaccount by name, alias to "DeleteByName"
func (s *ServiceAccount) Delete(name string) (err error) {
	return s.DeleteByName(name)
}

// GetFromBytes get serviceaccount from bytes
func (s *ServiceAccount) GetFromBytes(data []byte) (*corev1.ServiceAccount, error) {
	saJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	sa := &corev1.ServiceAccount{}
	err = json.Unmarshal(saJson, sa)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(sa.Namespace) != 0 {
		namespace = sa.Namespace
	} else {
		namespace = s.namespace
	}

	return s.WithNamespace(namespace).GetByName(sa.Name)
}

// GetFromFile get serviceaccount from yaml file
func (s *ServiceAccount) GetFromFile(path string) (*corev1.ServiceAccount, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.GetFromBytes(data)
}

// GetByName get serviceaccount by name
func (s *ServiceAccount) GetByName(name string) (*corev1.ServiceAccount, error) {
	return s.clientset.CoreV1().ServiceAccounts(s.namespace).Get(s.ctx, name, s.Options.GetOptions)
}

// Get get serviceaccount by name
func (s *ServiceAccount) Get(name string) (*corev1.ServiceAccount, error) {
	return s.GetByName(name)
}

// ListByLabel list serviceaccounts by labels
func (s *ServiceAccount) ListByLabel(labels string) (*corev1.ServiceAccountList, error) {
	listOptions := s.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return s.clientset.CoreV1().ServiceAccounts(s.namespace).List(s.ctx, *listOptions)
}

// List list serviceaccounts by labels, alias to "ListByLabel"
func (s *ServiceAccount) List(labels string) (*corev1.ServiceAccountList, error) {
	return s.ListByLabel(labels)
}

// ListByNamespace list serviceaccounts by namespace
func (s *ServiceAccount) ListByNamespace(namespace string) (*corev1.ServiceAccountList, error) {
	return s.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all serviceaccounts in the k8s cluster
func (s *ServiceAccount) ListAll(namespace string) (*corev1.ServiceAccountList, error) {
	return s.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// WatchByName watch serviceaccounts by name
func (s *ServiceAccount) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: s.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = s.clientset.CoreV1().ServiceAccounts(s.namespace).Watch(s.ctx, listOptions); err != nil {
			logrus.Error(err)
			return
		}
		if _, err = s.Get(name); err != nil {
			isExist = false // serviceaccount not exist
		} else {
			isExist = true // serviceaccount exist
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
				log.Debug("watch serviceaccount: bookmark.")
			case watch.Error:
				log.Debug("watch serviceaccount: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch serviceaccount: reconnect to kubernetes")
	}
}

// WatchByLabel watch serviceaccounts by labelSelector
func (s *ServiceAccount) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher            watch.Interface
		serviceaccountList *corev1.ServiceAccountList
		timeout            = int64(0)
		isExist            bool
	)
	for {
		if watcher, err = s.clientset.CoreV1().ServiceAccounts(s.namespace).Watch(s.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			logrus.Error(err)
			return
		}
		if serviceaccountList, err = s.List(labelSelector); err != nil {
			logrus.Error(err)
			return
		}
		if len(serviceaccountList.Items) == 0 {
			isExist = false // serviceaccount not exist
		} else {
			isExist = true // serviceaccount exist
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
				log.Debug("watch serviceaccount: bookmark.")
			case watch.Error:
				log.Debug("watch serviceaccount: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch serviceaccount: reconnect to kubernetes")
	}
}

// Watch watch serviceaccounts by name, alias to "WatchByName"
func (s *ServiceAccount) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return s.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
