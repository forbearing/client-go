package k8s

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"

	log "github.com/sirupsen/logrus"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type StorageClass struct {
	kubeconfig string

	ctx             context.Context
	config          *rest.Config
	restClient      *rest.RESTClient
	clientset       *kubernetes.Clientset
	dynamicClient   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient

	Options *HandlerOptions

	sync.Mutex
}

// new a StorageClass handler from kubeconfig or in-cluster config
func NewStorageClass(ctx context.Context, kubeconfig string) (sc *StorageClass, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
	)
	sc = &StorageClass{}

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
	config.GroupVersion = &storagev1.SchemeGroupVersion
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

	sc.kubeconfig = kubeconfig

	sc.ctx = ctx
	sc.config = config
	sc.restClient = restClient
	sc.clientset = clientset
	sc.dynamicClient = dynamicClient
	sc.discoveryClient = discoveryClient

	sc.Options = &HandlerOptions{}

	return
}
func (in *StorageClass) DeepCopy() *StorageClass {
	out := new(StorageClass)

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
func (s *StorageClass) WithDryRun() *StorageClass {
	sc := s.DeepCopy()
	sc.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	sc.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	sc.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	sc.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	sc.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return sc
}
func (s *StorageClass) SetTimeout(timeout int64) {
	s.Lock()
	defer s.Unlock()
	s.Options.ListOptions.TimeoutSeconds = &timeout
}
func (s *StorageClass) SetLimit(limit int64) {
	s.Lock()
	defer s.Unlock()
	s.Options.ListOptions.Limit = limit
}
func (s *StorageClass) SetForceDelete(force bool) {
	s.Lock()
	defer s.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		s.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		s.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// create storageclass from bytes
func (s *StorageClass) CreateFromBytes(data []byte) (*storagev1.StorageClass, error) {
	scJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	sc := &storagev1.StorageClass{}
	if err = json.Unmarshal(scJson, sc); err != nil {
		return nil, err
	}

	return s.clientset.StorageV1().StorageClasses().Create(s.ctx, sc, s.Options.CreateOptions)
}

// create storageclass from file
func (s *StorageClass) CreateFromFile(path string) (*storagev1.StorageClass, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.CreateFromBytes(data)
}

// create storageclass from file, alias to "CreateFromFile"
func (s *StorageClass) Create(path string) (*storagev1.StorageClass, error) {
	return s.CreateFromFile(path)
}

// update storageclass from bytes
func (s *StorageClass) UpdateFromBytes(data []byte) (*storagev1.StorageClass, error) {
	scJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	sc := &storagev1.StorageClass{}
	err = json.Unmarshal(scJson, sc)
	if err != nil {
		return nil, err
	}

	return s.clientset.StorageV1().StorageClasses().Update(s.ctx, sc, s.Options.UpdateOptions)
}

// update storageclass from file
func (s *StorageClass) UpdateFromFile(path string) (*storagev1.StorageClass, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.UpdateFromBytes(data)
}

// update storageclass from file, alias to "UpdateFromFile"
func (s *StorageClass) Update(path string) (*storagev1.StorageClass, error) {
	return s.UpdateFromFile(path)
}

// apply storageclass from bytes
func (s *StorageClass) ApplyFromBytes(data []byte) (sc *storagev1.StorageClass, err error) {
	sc, err = s.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		sc, err = s.UpdateFromBytes(data)
	}
	return
}

// apply storageclass from file
func (s *StorageClass) ApplyFromFile(path string) (sc *storagev1.StorageClass, err error) {
	sc, err = s.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		sc, err = s.UpdateFromFile(path)
	}
	return
}

// apply storageclass from file, alias to "ApplyFromFile"
func (s *StorageClass) Apply(path string) (*storagev1.StorageClass, error) {
	return s.ApplyFromFile(path)
}

// delete storageclass from bytes
func (s *StorageClass) DeleteFromBytes(data []byte) error {
	scJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	sc := &storagev1.StorageClass{}
	if err = json.Unmarshal(scJson, sc); err != nil {
		return err
	}

	return s.DeleteByName(sc.Name)
}

// delete storageclass from file
func (s *StorageClass) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return s.DeleteFromBytes(data)
}

// delete storageclass by name
func (s *StorageClass) DeleteByName(name string) error {
	return s.clientset.StorageV1().StorageClasses().Delete(s.ctx, name, s.Options.DeleteOptions)
}

// delete storageclass by name, alias to "DeleteByName"
func (s *StorageClass) Delete(name string) error {
	return s.DeleteByName(name)
}

// get storageclass from bytes
func (s *StorageClass) GetFromBytes(data []byte) (*storagev1.StorageClass, error) {
	scJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	sc := &storagev1.StorageClass{}
	err = json.Unmarshal(scJson, sc)
	if err != nil {
		return nil, err
	}

	return s.GetByName(sc.Name)
}

// get storageclass from file
func (s *StorageClass) GetFromFile(path string) (*storagev1.StorageClass, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.GetFromBytes(data)
}

// get storageclass by name
func (s *StorageClass) GetByName(name string) (*storagev1.StorageClass, error) {
	return s.clientset.StorageV1().StorageClasses().Get(s.ctx, name, s.Options.GetOptions)
}

// get storageclass by name, alias to "GetByName
func (s *StorageClass) Get(name string) (*storagev1.StorageClass, error) {
	return s.GetByName(name)
}

// ListByLabel list storageclasses by labels
func (s *StorageClass) ListByLabel(labels string) (*storagev1.StorageClassList, error) {
	listOptions := s.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return s.clientset.StorageV1().StorageClasses().List(s.ctx, *listOptions)
}

// List list storageclasses by labels, alias to "ListByLabel"
func (s *StorageClass) List(labels string) (*storagev1.StorageClassList, error) {
	return s.ListByLabel(labels)
}

// ListAll list all storageclasses in the k8s cluster
func (s *StorageClass) ListAll() (*storagev1.StorageClassList, error) {
	return s.ListByLabel("")
}

// watch storageclass by name
func (s *StorageClass) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = s.clientset.StorageV1().StorageClasses().Watch(s.ctx, listOptions); err != nil {
			return
		}
		if _, err = s.Get(name); err != nil {
			isExist = false // sc not exist
		} else {
			isExist = true // sc exist
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
				log.Debug("watch storageclass: bookmark.")
			case watch.Error:
				log.Debug("watch storageclass: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch storageclass: reconnect to kubernetes")
	}
}

// watch storageclass by labelSelector
func (s *StorageClass) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		scList  *storagev1.StorageClassList
		timeout = int64(0)
		isExist bool
	)
	for {
		if watcher, err = s.clientset.StorageV1().StorageClasses().Watch(s.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if scList, err = s.List(labelSelector); err != nil {
			return
		}
		if len(scList.Items) == 0 {
			isExist = false // sc not exist
		} else {
			isExist = true // sc exist
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
				log.Debug("watch storageclass: bookmark.")
			case watch.Error:
				log.Debug("watch storageclass: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch storageclass: reconnect to kubernetes")
	}
}

// watch storageclass by name, alias to "WatchByName"
func (s *StorageClass) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return s.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
