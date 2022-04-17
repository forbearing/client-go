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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type PersistentVolume struct {
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

// new a PersistentVolume handler from kubeconfig or in-cluster config
func NewPersistentVolume(ctx context.Context, kubeconfig string) (pv *PersistentVolume, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
		informerFactory informers.SharedInformerFactory
	)
	pv = &PersistentVolume{}

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

	pv.kubeconfig = kubeconfig
	pv.ctx = ctx
	pv.config = config
	pv.restClient = restClient
	pv.clientset = clientset
	pv.dynamicClient = dynamicClient
	pv.discoveryClient = discoveryClient
	pv.informerFactory = informerFactory
	pv.informer = informerFactory.Core().V1().PersistentVolumes().Informer()
	pv.Options = &HandlerOptions{}

	return
}
func (in *PersistentVolume) DeepCopy() *PersistentVolume {
	out := new(PersistentVolume)

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
func (p *PersistentVolume) WithDryRun() *PersistentVolume {
	pv := p.DeepCopy()
	pv.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	pv.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	pv.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	pv.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	pv.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return pv
}
func (p *PersistentVolume) SetTimeout(timeout int64) {
	p.Lock()
	defer p.Unlock()
	p.Options.ListOptions.TimeoutSeconds = &timeout
}
func (p *PersistentVolume) SetLimit(limit int64) {
	p.Lock()
	defer p.Unlock()
	p.Options.ListOptions.Limit = limit
}
func (p *PersistentVolume) SetForceDelete(force bool) {
	p.Lock()
	defer p.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		p.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		p.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// CreateFromRaw create persistentvolume from map[string]interface{}
func (p *PersistentVolume) CreateFromRaw(raw map[string]interface{}) (*corev1.PersistentVolume, error) {
	pv := &corev1.PersistentVolume{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, pv)
	if err != nil {
		return nil, err
	}

	return p.clientset.CoreV1().PersistentVolumes().Create(p.ctx, pv, p.Options.CreateOptions)
}

// CreateFromBytes create persistentvolume from bytes
func (p *PersistentVolume) CreateFromBytes(data []byte) (*corev1.PersistentVolume, error) {
	pvJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	pv := &corev1.PersistentVolume{}
	if err = json.Unmarshal(pvJson, pv); err != nil {
		return nil, err
	}

	return p.clientset.CoreV1().PersistentVolumes().Create(p.ctx, pv, p.Options.CreateOptions)
}

// CreateFromFile create persistentvolume from yaml file
func (p *PersistentVolume) CreateFromFile(path string) (*corev1.PersistentVolume, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p.CreateFromBytes(data)
}

// Create create persistentvolume from yaml file, alias to "CreateFromFile"
func (p *PersistentVolume) Create(path string) (*corev1.PersistentVolume, error) {
	return p.CreateFromFile(path)
}

// UpdateFromRaw update persistentvolume from map[string]interface{}
func (p *PersistentVolume) UpdateFromRaw(raw map[string]interface{}) (*corev1.PersistentVolume, error) {
	pv := &corev1.PersistentVolume{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, pv)
	if err != nil {
		return nil, err
	}

	return p.clientset.CoreV1().PersistentVolumes().Update(p.ctx, pv, p.Options.UpdateOptions)
}

// UpdateFromBytes update persistentvolume from bytes
func (p *PersistentVolume) UpdateFromBytes(data []byte) (*corev1.PersistentVolume, error) {
	pvJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	pv := &corev1.PersistentVolume{}
	err = json.Unmarshal(pvJson, pv)
	if err != nil {
		return nil, err
	}

	return p.clientset.CoreV1().PersistentVolumes().Update(p.ctx, pv, p.Options.UpdateOptions)
}

// UpdateFromFile update persistentvolume from yaml file
func (p *PersistentVolume) UpdateFromFile(path string) (*corev1.PersistentVolume, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p.UpdateFromBytes(data)
}

// Update update persistentvolume from yaml file, alias to "UpdateFromFile"
func (p *PersistentVolume) Update(path string) (*corev1.PersistentVolume, error) {
	return p.UpdateFromFile(path)
}

// ApplyFromRaw apply persistentvolume from map[string]interface{}
func (p *PersistentVolume) ApplyFromRaw(raw map[string]interface{}) (*corev1.PersistentVolume, error) {
	pv := &corev1.PersistentVolume{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, pv)
	if err != nil {
		return nil, err
	}

	pv, err = p.clientset.CoreV1().PersistentVolumes().Create(p.ctx, pv, p.Options.CreateOptions)
	if k8serrors.IsAlreadyExists(err) {
		pv, err = p.clientset.CoreV1().PersistentVolumes().Update(p.ctx, pv, p.Options.UpdateOptions)
	}
	return pv, err
}

// ApplyFromBytes apply persistentvolume from bytes
func (p *PersistentVolume) ApplyFromBytes(data []byte) (pv *corev1.PersistentVolume, err error) {
	pv, err = p.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		pv, err = p.UpdateFromBytes(data)
	}
	return
}

// ApplyFromFile apply persistentvolume from yaml file
func (p *PersistentVolume) ApplyFromFile(path string) (pv *corev1.PersistentVolume, err error) {
	pv, err = p.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		pv, err = p.UpdateFromFile(path)
	}
	return
}

// Apply apply persistentvolume from yaml file, alias to "ApplyFromFile"
func (p *PersistentVolume) Apply(path string) (*corev1.PersistentVolume, error) {
	return p.ApplyFromFile(path)
}

// DeleteFromBytes delete persistentvolume from bytes
func (p *PersistentVolume) DeleteFromBytes(data []byte) error {
	pvJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	pv := &corev1.PersistentVolume{}
	if err = json.Unmarshal(pvJson, pv); err != nil {
		return err
	}

	return p.DeleteByName(pv.Name)
}

// DeleteFromFile delete persistentvolume from yaml file
func (p *PersistentVolume) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return p.DeleteFromBytes(data)
}

// DeleteByName delete persistentvolume by name
func (p *PersistentVolume) DeleteByName(name string) error {
	return p.clientset.CoreV1().PersistentVolumes().Delete(p.ctx, name, p.Options.DeleteOptions)
}

// Delete delete persistentvolume by name, alias to "DeleteByName"
func (p *PersistentVolume) Delete(name string) error {
	return p.DeleteByName(name)
}

// GetFromBytes get persistentvolume from bytes
func (p *PersistentVolume) GetFromBytes(data []byte) (*corev1.PersistentVolume, error) {
	pvJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	pv := &corev1.PersistentVolume{}
	err = json.Unmarshal(pvJson, pv)
	if err != nil {
		return nil, err
	}

	return p.GetByName(pv.Name)
}

// GetFromFile get persistentvolume from yaml file
func (p *PersistentVolume) GetFromFile(path string) (*corev1.PersistentVolume, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p.GetFromBytes(data)
}

// GetByName get persistentvolume by name
func (p *PersistentVolume) GetByName(name string) (*corev1.PersistentVolume, error) {
	return p.clientset.CoreV1().PersistentVolumes().Get(p.ctx, name, p.Options.GetOptions)
}

// Get get persistentvolume by name, alias to "GetByName
func (p *PersistentVolume) Get(name string) (*corev1.PersistentVolume, error) {
	return p.GetByName(name)
}

// ListByLabel list persistentvolumes by labels
func (p *PersistentVolume) ListByLabel(labels string) (*corev1.PersistentVolumeList, error) {
	listOptions := p.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return p.clientset.CoreV1().PersistentVolumes().List(p.ctx, *listOptions)
}

// List list persistentvolumes by labels, alias to "ListByLabel"
func (p *PersistentVolume) List(labels string) (*corev1.PersistentVolumeList, error) {
	return p.ListByLabel(labels)
}

// ListAll list all persistentvolumes in the k8s cluster
func (p *PersistentVolume) ListAll() (*corev1.PersistentVolumeList, error) {
	return p.ListByLabel("")
}

// GetPVC get the pvc name of the persistentvolume
func (p *PersistentVolume) GetPVC(name string) (pvc string, err error) {
	pv, err := p.Get(name)
	if err != nil {
		return
	}
	if pv.Spec.ClaimRef != nil {
		if pv.Spec.ClaimRef.Kind == "PersistentVolumeClaim" {
			pvc = pv.Spec.ClaimRef.Name
		}
	}
	return
}

// GetStorageClass get the storageclass name of the persistentvolume
func (p *PersistentVolume) GetStorageClass(name string) (sc string, err error) {
	pv, err := p.Get(name)
	if err != nil {
		return
	}
	sc = pv.Spec.StorageClassName
	return
}

// GetAccessModes get the accessModes of the persistentvolume
func (p *PersistentVolume) GetAccessModes(name string) (accessModes []string, err error) {
	pv, err := p.Get(name)
	if err != nil {
		return
	}
	for _, accessMode := range pv.Spec.AccessModes {
		accessModes = append(accessModes, string(accessMode))
	}
	return
}

// GetCapacity get the the storage capacity of the persistentvolume
func (p *PersistentVolume) GetCapacity(name string) (capacity int64, err error) {
	pv, err := p.Get(name)
	if err != nil {
		return
	}
	storage := pv.Spec.Capacity[corev1.ResourceName(corev1.ResourceStorage)]
	//capacity = storage.Value()
	//capacity = storage.MilliValue()
	//capacity = storage.ScaledValue(resource.Kilo)
	//capacity = storage.ScaledValue(resource.Mega)
	//capacity = storage.ScaledValue(resource.Giga)
	//capacity = storage.ScaledValue(resource.Tera)
	//capacity = storage.ScaledValue(resource.Peta)
	//capacity = storage.ScaledValue(resource.Exa)
	capacity = storage.Value()
	return
}

// GetPhase get the status phase of the persistentvolume
func (p *PersistentVolume) GetPhase(name string) (phase string, err error) {
	pv, err := p.Get(name)
	if err != nil {
		return
	}
	phase = string(pv.Status.Phase)
	return
}

// GetReclaimPolicy get the reclaim policy of the persistentvolume
func (p *PersistentVolume) GetReclaimPolicy(name string) (policy string, err error) {
	pv, err := p.Get(name)
	if err != nil {
		return
	}
	policy = string(pv.Spec.PersistentVolumeReclaimPolicy)
	return
}

// WatchByName watch persistentvolume by name
func (p *PersistentVolume) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = p.clientset.CoreV1().PersistentVolumes().Watch(p.ctx, listOptions); err != nil {
			return
		}
		if _, err = p.Get(name); err != nil {
			isExist = false // pv not exist
		} else {
			isExist = true // pv exist
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
				log.Debug("watch persistentvolume: bookmark.")
			case watch.Error:
				log.Debug("watch persistentvolume: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch persistentvolume: reconnect to kubernetes")
	}
}

// WatchByLabel watch persistentvolume by labelSelector
func (p *PersistentVolume) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		pvList  *corev1.PersistentVolumeList
		timeout = int64(0)
		isExist bool
	)
	for {
		if watcher, err = p.clientset.CoreV1().PersistentVolumes().Watch(p.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if pvList, err = p.List(labelSelector); err != nil {
			return
		}
		if len(pvList.Items) == 0 {
			isExist = false // pv not exist
		} else {
			isExist = true // pv exist
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
				log.Debug("watch persistentvolume: bookmark.")
			case watch.Error:
				log.Debug("watch persistentvolume: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch persistentvolume: reconnect to kubernetes")
	}
}

// Watch watch persistentvolume by name, alias to "WatchByName"
func (p *PersistentVolume) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return p.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}

// RunInformer
func (p *PersistentVolume) RunInformer(
	addFunc func(obj interface{}),
	updateFunc func(oldObj, newObj interface{}),
	deleteFunc func(obj interface{}),
	stopCh chan struct{}) {
	p.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    addFunc,
		UpdateFunc: updateFunc,
		DeleteFunc: deleteFunc,
	})
	p.informer.Run(stopCh)
}
