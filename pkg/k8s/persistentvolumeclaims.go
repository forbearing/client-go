package k8s

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
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

type PersistentVolumeClaim struct {
	kubeconfig string
	namespace  string

	ctx             context.Context
	config          *rest.Config
	restClient      *rest.RESTClient
	clientset       *kubernetes.Clientset
	dynamicClient   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient

	Options *HandlerOptions

	sync.Mutex
}

// new a PersistentVolumeClaim handler from kubeconfig or in-cluster config
func NewPersistentVolumeClaim(ctx context.Context, namespace, kubeconfig string) (pvc *PersistentVolumeClaim, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
	)
	pvc = &PersistentVolumeClaim{}

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
	if len(namespace) == 0 {
		namespace = metav1.NamespaceDefault
	}
	pvc.kubeconfig = kubeconfig
	pvc.namespace = namespace

	pvc.ctx = ctx
	pvc.config = config
	pvc.restClient = restClient
	pvc.clientset = clientset
	pvc.dynamicClient = dynamicClient
	pvc.discoveryClient = discoveryClient

	pvc.Options = &HandlerOptions{}

	return
}
func (p *PersistentVolumeClaim) Namespace() string {
	return p.namespace
}
func (in *PersistentVolumeClaim) DeepCopy() *PersistentVolumeClaim {
	out := new(PersistentVolumeClaim)

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
func (p *PersistentVolumeClaim) setNamespace(namespace string) {
	p.Lock()
	defer p.Unlock()
	p.namespace = namespace
}
func (p *PersistentVolumeClaim) WithNamespace(namespace string) *PersistentVolumeClaim {
	pvc := p.DeepCopy()
	pvc.setNamespace(namespace)
	return pvc
}
func (p *PersistentVolumeClaim) WithDryRun() *PersistentVolumeClaim {
	pvc := p.DeepCopy()
	pvc.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	pvc.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	pvc.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	pvc.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	pvc.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return pvc
}
func (p *PersistentVolumeClaim) SetTimeout(timeout int64) {
	p.Lock()
	defer p.Unlock()
	p.Options.ListOptions.TimeoutSeconds = &timeout
}
func (p *PersistentVolumeClaim) SetLimit(limit int64) {
	p.Lock()
	defer p.Unlock()
	p.Options.ListOptions.Limit = limit
}
func (p *PersistentVolumeClaim) SetForceDelete(force bool) {
	p.Lock()
	defer p.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		p.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		p.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// create persistentvolumeclaim from bytes
func (p *PersistentVolumeClaim) CreateFromBytes(data []byte) (*corev1.PersistentVolumeClaim, error) {
	pvcJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	pvc := &corev1.PersistentVolumeClaim{}
	err = json.Unmarshal(pvcJson, pvc)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(pvc.Namespace) != 0 {
		namespace = pvc.Namespace
	} else {
		namespace = p.namespace
	}

	return p.clientset.CoreV1().PersistentVolumeClaims(namespace).Create(p.ctx, pvc, p.Options.CreateOptions)
}

// create persistentvolumeclaim from file
func (p *PersistentVolumeClaim) CreateFromFile(path string) (*corev1.PersistentVolumeClaim, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p.CreateFromBytes(data)
}

// create persistentvolumeclaim from file,alias to "CreateFromFile"
func (p *PersistentVolumeClaim) Create(path string) (*corev1.PersistentVolumeClaim, error) {
	return p.CreateFromFile(path)
}

// update persistentvolumeclaim from bytes
func (p *PersistentVolumeClaim) UpdateFromBytes(data []byte) (*corev1.PersistentVolumeClaim, error) {
	pvcJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	pvc := &corev1.PersistentVolumeClaim{}
	err = json.Unmarshal(pvcJson, pvc)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(pvc.Namespace) != 0 {
		namespace = pvc.Namespace
	} else {
		namespace = p.namespace
	}

	return p.clientset.CoreV1().PersistentVolumeClaims(namespace).Update(p.ctx, pvc, p.Options.UpdateOptions)
}

// update persistentvolumeclaim from file
func (p *PersistentVolumeClaim) UpdateFromFile(path string) (*corev1.PersistentVolumeClaim, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p.UpdateFromBytes(data)
}

// update persistentvolumeclaim from file, alias to "UpdateFromFile"
func (p *PersistentVolumeClaim) Update(path string) (*corev1.PersistentVolumeClaim, error) {
	return p.UpdateFromFile(path)
}

// apply persistentvolumeclaim from bytes
func (p *PersistentVolumeClaim) ApplyFromBytes(data []byte) (pvc *corev1.PersistentVolumeClaim, err error) {
	pvc, err = p.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		pvc, err = p.UpdateFromBytes(data)
	}
	return
}

// apply persistentvolumeclaim from file
func (p *PersistentVolumeClaim) ApplyFromFile(path string) (pvc *corev1.PersistentVolumeClaim, err error) {
	pvc, err = p.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		pvc, err = p.UpdateFromFile(path)
	}
	return
}

// apply persistentvolumeclaim from file, alias to "ApplyFromFile"
func (p *PersistentVolumeClaim) Apply(path string) (*corev1.PersistentVolumeClaim, error) {
	return p.ApplyFromFile(path)
}

// delete persistentvolumeclaim from bytes
func (p *PersistentVolumeClaim) DeleteFromBytes(data []byte) error {
	pvcJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	pvc := &corev1.PersistentVolumeClaim{}
	if err = json.Unmarshal(pvcJson, pvc); err != nil {
		return err
	}

	var namespace string
	if len(pvc.Namespace) != 0 {
		namespace = pvc.Namespace
	} else {
		namespace = p.namespace
	}

	return p.WithNamespace(namespace).DeleteByName(pvc.Name)
}

// delete persistentvolumeclaim from file
func (p *PersistentVolumeClaim) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return p.DeleteFromBytes(data)
}

// delete persistentvolumeclaim by name
func (p *PersistentVolumeClaim) DeleteByName(name string) error {
	return p.clientset.CoreV1().PersistentVolumeClaims(p.namespace).Delete(p.ctx, name, p.Options.DeleteOptions)
}

// delete persistentvolumeclaim by name, alias to "DeleteByName"
func (p *PersistentVolumeClaim) Delete(name string) error {
	return p.DeleteByName(name)
}

// list persistentvolumeclaim by labelSelector
func (p *PersistentVolumeClaim) List(labelSelector string) (*corev1.PersistentVolumeClaimList, error) {
	p.Options.ListOptions.LabelSelector = labelSelector
	return p.clientset.CoreV1().PersistentVolumeClaims(p.namespace).List(p.ctx, p.Options.ListOptions)
}

// get persistentvolumeclaim from bytes
func (p *PersistentVolumeClaim) GetFromBytes(data []byte) (*corev1.PersistentVolumeClaim, error) {
	pvcJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	pvc := &corev1.PersistentVolumeClaim{}
	if err = json.Unmarshal(pvcJson, pvc); err != nil {
		return nil, err
	}

	var namespace string
	if len(pvc.Namespace) != 0 {
		namespace = pvc.Namespace
	} else {
		namespace = p.namespace
	}

	return p.WithNamespace(namespace).GetByName(pvc.Name)
}

// get persistentvolumeclaim from file
func (p *PersistentVolumeClaim) GetFromFile(path string) (*corev1.PersistentVolumeClaim, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p.GetFromBytes(data)
}

// get persistentvolumeclaim by name
func (p *PersistentVolumeClaim) GetByName(name string) (*corev1.PersistentVolumeClaim, error) {
	return p.clientset.CoreV1().PersistentVolumeClaims(p.namespace).Get(p.ctx, name, p.Options.GetOptions)
}

// get persistentvolumeclaim by name, alias to "GetByName"
func (p *PersistentVolumeClaim) Get(name string) (*corev1.PersistentVolumeClaim, error) {
	return p.GetByName(name)
}

// get the pv name of the persistentvolumeclaim
func (p *PersistentVolumeClaim) GetPV(name string) (pv string, err error) {
	pvc, err := p.Get(name)
	if err != nil {
		return
	}
	pv = pvc.Spec.VolumeName
	return
}
func (p *PersistentVolumeClaim) GetStorageClass(name string) (sc string, err error) {
	pvc, err := p.Get(name)
	if err != nil {
		return
	}
	sc = *pvc.Spec.StorageClassName
	return
}

// get the access modes of the persistentvolumeclaim
func (p *PersistentVolumeClaim) GetAccessModes(name string) (accessModes []string, err error) {
	pvc, err := p.Get(name)
	if err != nil {
		return
	}
	for _, accessMode := range pvc.Status.AccessModes {
		accessModes = append(accessModes, string(accessMode))
	}
	return
}

// get the storage capacity of the persistentvolumeclaim
func (p *PersistentVolumeClaim) GetCapacity(name string) (capacity int64, err error) {
	pvc, err := p.Get(name)
	if err != nil {
		return
	}
	storage := pvc.Status.Capacity[corev1.ResourceName(corev1.ResourceStorage)]
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

// get the status phase of the persistentvolumeclaim
func (p *PersistentVolumeClaim) GetPhase(name string) (phase string, err error) {
	pvc, err := p.Get(name)
	if err != nil {
		return
	}
	phase = string(pvc.Status.Phase)
	return
}

// watch persistentvolumeclaim by name
func (p *PersistentVolumeClaim) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: p.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = p.clientset.CoreV1().PersistentVolumeClaims(p.namespace).Watch(p.ctx, listOptions); err != nil {
			return
		}
		if _, err = p.Get(name); err != nil {
			isExist = false // pvc not exist
		} else {
			isExist = true // pvc exist
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
				log.Debug("watch persistentvolumeclaim: bookmark.")
			case watch.Error:
				log.Debug("watch persistentvolumeclaim: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch persistentvolumeclaim: reconnect to kubernetes")
	}
}

// watch persistentvolumeclaim by labelSelector
func (p *PersistentVolumeClaim) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		pvcList *corev1.PersistentVolumeClaimList
		timeout = int64(0)
		isExist bool
	)
	for {
		if watcher, err = p.clientset.CoreV1().PersistentVolumeClaims(p.namespace).Watch(p.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if pvcList, err = p.List(labelSelector); err != nil {
			return
		}
		if len(pvcList.Items) == 0 {
			isExist = false // pvc not exist
		} else {
			isExist = true // pvc exist
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
				log.Debug("watch persistentvolumeclaim: bookmark.")
			case watch.Error:
				log.Debug("watch persistentvolumeclaim: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch persistentvolumeclaim: reconnect to kubernetes")
	}
}

// watch persistentvolumeclaim by name, alias to "WatchByName"
func (p *PersistentVolumeClaim) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return p.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}