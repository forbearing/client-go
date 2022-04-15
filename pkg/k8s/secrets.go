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

type Secret struct {
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

// new a Secret handler from kubeconfig or in-cluster config
func NewSecret(ctx context.Context, namespace, kubeconfig string) (secret *Secret, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	secret = &Secret{}

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
	secret.kubeconfig = kubeconfig
	secret.namespace = namespace
	secret.ctx = ctx
	secret.config = config
	secret.restClient = restClient
	secret.clientset = clientset
	secret.dynamicClient = dynamicClient
	secret.discoveryClient = discoveryClient
	secret.informerFactory = informerFactory
	secret.Options = &HandlerOptions{}

	return
}
func (s *Secret) Namespace() string {
	return s.namespace
}
func (in *Secret) DeepCopy() *Secret {
	out := new(Secret)

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
func (s *Secret) setNamespace(namespace string) {
	s.Lock()
	defer s.Unlock()
	s.namespace = namespace
}
func (s *Secret) WithNamespace(namespace string) *Secret {
	secret := s.DeepCopy()
	secret.setNamespace(namespace)
	return secret
}
func (s *Secret) WithDryRun() *Secret {
	secret := s.DeepCopy()
	secret.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	secret.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	secret.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	secret.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	secret.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return secret
}
func (s *Secret) SetTimeout(timeout int64) {
	s.Lock()
	defer s.Unlock()
	s.Options.ListOptions.TimeoutSeconds = &timeout
}
func (s *Secret) SetLimit(limit int64) {
	s.Lock()
	defer s.Unlock()
	s.Options.ListOptions.Limit = limit
}
func (s *Secret) SetForceDelete(force bool) {
	s.Lock()
	defer s.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		s.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		s.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create secret from map[string]interface{}
func (s *Secret) CreateFromRaw(raw map[string]interface{}) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, secret)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(secret.Namespace) != 0 {
		namespace = secret.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.CoreV1().Secrets(namespace).Create(s.ctx, secret, s.Options.CreateOptions)
}

// CreateFromBytes create secret from bytes
func (s *Secret) CreateFromBytes(data []byte) (*corev1.Secret, error) {
	secretJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}
	err = json.Unmarshal(secretJson, secret)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(secret.Namespace) != 0 {
		namespace = secret.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.CoreV1().Secrets(namespace).Create(s.ctx, secret, s.Options.CreateOptions)
}

// CreateFromFile create secret from yaml file
func (s *Secret) CreateFromFile(path string) (*corev1.Secret, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.CreateFromBytes(data)
}

// Create create secret from yaml file, alias to "CreateFromFile"
func (s *Secret) Create(path string) (*corev1.Secret, error) {
	return s.CreateFromFile(path)
}

// UpdateFromRaw update secret from map[string]interface{}
func (s *Secret) UpdateFromRaw(raw map[string]interface{}) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, secret)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(secret.Namespace) != 0 {
		namespace = secret.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.CoreV1().Secrets(namespace).Update(s.ctx, secret, s.Options.UpdateOptions)
}

// UpdateFromBytes update secret from bytes
func (s *Secret) UpdateFromBytes(data []byte) (*corev1.Secret, error) {
	secretJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}
	err = json.Unmarshal(secretJson, secret)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(secret.Namespace) != 0 {
		namespace = secret.Namespace
	} else {
		namespace = s.namespace
	}

	return s.clientset.CoreV1().Secrets(namespace).Update(s.ctx, secret, s.Options.UpdateOptions)
}

// UpdateFromFile update secret from yaml file
func (s *Secret) UpdateFromFile(path string) (*corev1.Secret, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.UpdateFromBytes(data)
}

// Update update secret from yaml file, alias to "UpdateFromFile"
func (s *Secret) Update(path string) (*corev1.Secret, error) {
	return s.UpdateFromFile(path)
}

// ApplyFromRaw apply secret from map[string]interface{}
func (s *Secret) ApplyFromRaw(raw map[string]interface{}) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, secret)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(secret.Namespace) != 0 {
		namespace = secret.Namespace
	} else {
		namespace = s.namespace
	}

	secret, err = s.clientset.CoreV1().Secrets(namespace).Create(s.ctx, secret, s.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		secret, err = s.clientset.CoreV1().Secrets(namespace).Update(s.ctx, secret, s.Options.UpdateOptions)
	}
	return secret, err
}

// ApplyFromBytes apply secret from bytes
func (s *Secret) ApplyFromBytes(data []byte) (secret *corev1.Secret, err error) {
	secret, err = s.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		secret, err = s.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply secret from yaml file
func (s *Secret) ApplyFromFile(path string) (secret *corev1.Secret, err error) {
	secret, err = s.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		secret, err = s.UpdateFromFile(path)
	}
	return
}

// Apply apply secret from yaml file, alias to "ApplyFromFile"
func (s *Secret) Apply(path string) (*corev1.Secret, error) {
	return s.ApplyFromFile(path)
}

// DeleteFromBytes delete secret from bytes
func (s *Secret) DeleteFromBytes(data []byte) error {
	secretJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{}
	err = json.Unmarshal(secretJson, secret)
	if err != nil {
		return err
	}

	var namespace string
	if len(secret.Namespace) != 0 {
		namespace = secret.Namespace
	} else {
		namespace = s.namespace
	}

	return s.WithNamespace(namespace).DeleteByName(secret.Name)
}

// DeleteFromFile delete secret from yaml file
func (s *Secret) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return s.DeleteFromBytes(data)
}

// DeleteByName delete secret by name
func (s *Secret) DeleteByName(name string) error {
	return s.clientset.CoreV1().Secrets(s.namespace).Delete(s.ctx, name, s.Options.DeleteOptions)
}

// Delete delete secret by name, alias to "DeleteByName"
func (s *Secret) Delete(name string) error {
	return s.DeleteByName(name)
}

// GetFromBytes get secret from bytes
func (s *Secret) GetFromBytes(data []byte) (*corev1.Secret, error) {
	secretJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}
	err = json.Unmarshal(secretJson, secret)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(secret.Namespace) != 0 {
		namespace = secret.Namespace
	} else {
		namespace = s.namespace
	}

	return s.WithNamespace(namespace).GetByName(secret.Name)
}

// GetFromFile get secret from yaml file
func (s *Secret) GetFromFile(path string) (*corev1.Secret, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.GetFromBytes(data)
}

// GetByName get secret by name
func (s *Secret) GetByName(name string) (*corev1.Secret, error) {
	return s.clientset.CoreV1().Secrets(s.namespace).Get(s.ctx, name, s.Options.GetOptions)
}

// Get get secret by name, alias to "GetByName"
func (s *Secret) Get(name string) (*corev1.Secret, error) {
	return s.GetByName(name)
}

// ListByLabel list secrets by labels
func (s *Secret) ListByLabel(labels string) (*corev1.SecretList, error) {
	listOptions := s.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return s.clientset.CoreV1().Secrets(s.namespace).List(s.ctx, *listOptions)
}

// List list secrets by labels, alias to "ListByLabel"
func (s *Secret) List(labels string) (*corev1.SecretList, error) {
	return s.ListByLabel(labels)
}

// ListByNamespace list secrets by namespace
func (s *Secret) ListByNamespace(namespace string) (*corev1.SecretList, error) {
	return s.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all secrets in the k8s cluster
func (s *Secret) ListAll() (*corev1.SecretList, error) {
	return s.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// WatchByName watch secret by name
func (s *Secret) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: s.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = s.clientset.CoreV1().Secrets(s.namespace).Watch(s.ctx, listOptions); err != nil {
			logrus.Error(err)
			return
		}
		if _, err = s.Get(name); err != nil {
			isExist = false // secret not exist
		} else {
			isExist = true // secret exist
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
				log.Debug("watch secret: bookmark.")
			case watch.Error:
				log.Debug("watch secret: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch secret: reconnect to kubernetes")
	}
}

// WatchByLabel watch secret by labelSelector
func (s *Secret) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher    watch.Interface
		secretList *corev1.SecretList
		timeout    = int64(0)
		isExist    bool
	)
	for {
		if watcher, err = s.clientset.CoreV1().Secrets(s.namespace).Watch(s.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			logrus.Error(err)
			return
		}
		if secretList, err = s.List(labelSelector); err != nil {
			logrus.Error(err)
			return
		}
		if len(secretList.Items) == 0 {
			isExist = false // secret not exist
		} else {
			isExist = true // secret exist
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
				log.Debug("watch secret: bookmark.")
			case watch.Error:
				log.Debug("watch secret: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch secret: reconnect to kubernetes")
	}
}

// Watch watch secret by name, alias to "WatchByName"
func (s *Secret) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return s.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
