package k8s

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
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

type CronJob struct {
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

// new a cronjob handler from kubeconfig or in-cluster config
func NewCronJob(ctx context.Context, namespace, kubeconfig string) (cronjob *CronJob, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
	)
	cronjob = &CronJob{}

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
	config.GroupVersion = &batchv1.SchemeGroupVersion
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
	cronjob.kubeconfig = kubeconfig
	cronjob.namespace = namespace

	cronjob.ctx = ctx
	cronjob.config = config
	cronjob.restClient = restClient
	cronjob.clientset = clientset
	cronjob.dynamicClient = dynamicClient
	cronjob.discoveryClient = discoveryClient

	cronjob.Options = &HandlerOptions{}

	return
}
func (c *CronJob) Namespace() string {
	return c.namespace
}
func (in *CronJob) DeepCopy() *CronJob {
	out := new(CronJob)

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
func (c *CronJob) setNamespace(namespace string) {
	c.Lock()
	defer c.Unlock()
	c.namespace = namespace
}
func (c *CronJob) WithNamespace(namespace string) *CronJob {
	cj := c.DeepCopy()
	cj.setNamespace(namespace)
	return cj
}
func (c *CronJob) WithDryRun() *CronJob {
	cj := c.DeepCopy()
	cj.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	cj.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	cj.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	cj.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	cj.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	return cj
}
func (c *CronJob) SetTimeout(timeout int64) {
	c.Lock()
	defer c.Unlock()
	c.Options.ListOptions.TimeoutSeconds = &timeout
}
func (c *CronJob) SetLimit(limit int64) {
	c.Lock()
	defer c.Unlock()
	c.Options.ListOptions.Limit = limit
}
func (c *CronJob) SetForceDelete(force bool) {
	c.Lock()
	defer c.Lock()
	if force {
		gracePeriodSeconds := int64(0)
		c.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	} else {
		c.Options.DeleteOptions = metav1.DeleteOptions{}
	}
}

// Whether and how garbage collection will be performed.
// support value are "Background", "Orphan", "Foreground",
// default value is "Background"
func (c *CronJob) SetPropagationPolicy(policy string) {
	c.Lock()
	defer c.Unlock()
	switch strings.ToLower(policy) {
	case strings.ToLower(string(metav1.DeletePropagationBackground)):
		propagationPolicy := metav1.DeletePropagationBackground
		c.Options.DeleteOptions.PropagationPolicy = &propagationPolicy
	case strings.ToLower(string(metav1.DeletePropagationForeground)):
		propagationPolicy := metav1.DeletePropagationForeground
		c.Options.DeleteOptions.PropagationPolicy = &propagationPolicy
	case strings.ToLower(string(metav1.DeletePropagationOrphan)):
		propagationPolicy := metav1.DeletePropagationOrphan
		c.Options.DeleteOptions.PropagationPolicy = &propagationPolicy
	default:
		propagationPolicy := metav1.DeletePropagationBackground
		c.Options.DeleteOptions.PropagationPolicy = &propagationPolicy
	}
}

// create cronjob from bytes
func (c *CronJob) CreateFromBytes(data []byte) (*batchv1.CronJob, error) {
	cronjobJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	cronjob := &batchv1.CronJob{}
	err = json.Unmarshal(cronjobJson, cronjob)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(cronjob.Namespace) != 0 {
		namespace = cronjob.Namespace
	} else {
		namespace = c.namespace
	}

	return c.clientset.BatchV1().CronJobs(namespace).Create(c.ctx, cronjob, c.Options.CreateOptions)
}

// create cronjob from file
func (c *CronJob) CreateFromFile(path string) (*batchv1.CronJob, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.CreateFromBytes(data)
}

// create cronjob from file, alias to "CreateFromFile"
func (c *CronJob) Create(path string) (*batchv1.CronJob, error) {
	return c.CreateFromFile(path)
}

// update cronjob from bytes
func (c *CronJob) UpdateFromBytes(data []byte) (*batchv1.CronJob, error) {
	cronjobJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	cronjob := &batchv1.CronJob{}
	err = json.Unmarshal(cronjobJson, cronjob)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(cronjob.Namespace) != 0 {
		namespace = cronjob.Namespace
	} else {
		namespace = c.namespace
	}

	return c.clientset.BatchV1().CronJobs(namespace).Update(c.ctx, cronjob, c.Options.UpdateOptions)
}

// update cronjob from file
func (c *CronJob) UpdateFromFile(path string) (*batchv1.CronJob, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.UpdateFromBytes(data)
}

// update cronjob from file, alias to "UpdateFromFile"
func (c *CronJob) Update(path string) (*batchv1.CronJob, error) {
	return c.UpdateFromFile(path)
}

// apply cronjob from bytes
func (c *CronJob) ApplyFromBytes(data []byte) (cronjob *batchv1.CronJob, err error) {
	cronjob, err = c.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		cronjob, err = c.UpdateFromBytes(data)
	}
	return
}

// apply cronjob from file
func (c *CronJob) ApplyFromFile(path string) (cronjob *batchv1.CronJob, err error) {
	cronjob, err = c.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		cronjob, err = c.UpdateFromFile(path)
	}
	return
}

// apply cronjob from file, alias to "ApplyFromFile"
func (c *CronJob) Apply(path string) (*batchv1.CronJob, error) {
	return c.ApplyFromFile(path)
}

// delete cronjob from bytes
func (c *CronJob) DeleteFromBytes(data []byte) (err error) {
	cronjobJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	cronjob := &batchv1.CronJob{}
	err = json.Unmarshal(cronjobJson, cronjob)
	if err != nil {
		return err
	}

	var namespace string
	if len(cronjob.Namespace) != 0 {
		namespace = cronjob.Namespace
	} else {
		namespace = c.namespace
	}

	return c.clientset.BatchV1().CronJobs(namespace).Delete(c.ctx, cronjob.Name, c.Options.DeleteOptions)
}

// delete cronjob from file
func (c *CronJob) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return c.DeleteFromBytes(data)
}

// delete cronjob by name
func (c *CronJob) DeleteByName(name string) error {
	return c.clientset.BatchV1().CronJobs(c.namespace).Delete(c.ctx, name, c.Options.DeleteOptions)
}

// delete cronjob by name, alias to "DeleteByName"
func (c *CronJob) Delete(name string) (err error) {
	return c.DeleteByName(name)
}

// get cronjob from bytes
func (c *CronJob) GetFromBytes(data []byte) (*batchv1.CronJob, error) {
	cronjobJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	cronjob := &batchv1.CronJob{}
	if err = json.Unmarshal(cronjobJson, cronjob); err != nil {
		return nil, err
	}

	var namespace string
	if len(cronjob.Namespace) != 0 {
		namespace = cronjob.Namespace
	} else {
		namespace = c.namespace
	}

	return c.clientset.BatchV1().CronJobs(namespace).Get(c.ctx, cronjob.Name, c.Options.GetOptions)
}

// get cronjob from file
func (c *CronJob) GetFromFile(path string) (*batchv1.CronJob, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.GetFromBytes(data)
}

// get cronjob by name
func (c *CronJob) Get(name string) (*batchv1.CronJob, error) {
	return c.clientset.BatchV1().CronJobs(c.namespace).Get(c.ctx, name, c.Options.GetOptions)
}

// list cronjob by labelSelector
func (c *CronJob) List(labelSelector string) (*batchv1.CronJobList, error) {
	c.Options.ListOptions.LabelSelector = labelSelector
	return c.clientset.BatchV1().CronJobs(c.namespace).List(c.ctx, c.Options.ListOptions)
}

// watch cronjobs by name
func (c *CronJob) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOption := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: c.namespace})
		listOption.TimeoutSeconds = &timeout
		if watcher, err = c.clientset.BatchV1().CronJobs(c.namespace).Watch(c.ctx, listOption); err != nil {
			return
		}
		if _, err = c.Get(name); err != nil {
			isExist = false // cronjob not exist
		} else {
			isExist = true // cronjob exist
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
				log.Debug("watch cronjob: bookmark.")
			case watch.Error:
				log.Debug("watch cronjob: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch cronjob: reconnect to kubernetes")
	}
}

// watch cronjobs by labelSelector
func (c *CronJob) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher     watch.Interface
		cronjobList *batchv1.CronJobList
		timeout     = int64(0)
		isExist     bool
	)
	for {
		if watcher, err = c.clientset.BatchV1().CronJobs(c.namespace).Watch(c.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if cronjobList, err = c.List(labelSelector); err != nil {
			return
		}
		if len(cronjobList.Items) == 0 {
			isExist = false // cronjob not exist
		} else {
			isExist = true // cronjob exist
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
				log.Debug("watch cronjob: bookmark.")
			case watch.Error:
				log.Debug("watch cronjob: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch cronjob: reconnect to kubernetes")
	}
}

// watch cronjobs by name, alias to "WatchByName"
func (c *CronJob) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return c.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}