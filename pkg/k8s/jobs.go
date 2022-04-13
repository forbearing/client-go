package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
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

type JobController struct {
	Labels             map[string]string `json:"labels"`
	CreationTimestamp  metav1.Time       `json:"creationTimestamp"`
	LastScheduleTime   metav1.Time       `json:"lastScheduleTime"`
	LastSuccessfulTime metav1.Time       `json:"lastSuccessfulTime"`

	metav1.OwnerReference `json:"ownerReference"`
}
type Job struct {
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

// new a job handler from kubeconfig or in-cluster config
func NewJob(ctx context.Context, namespace, kubeconfig string) (job *Job, err error) {
	var (
		config          *rest.Config
		restClient      *rest.RESTClient
		clientset       *kubernetes.Clientset
		dynamicClient   dynamic.Interface
		discoveryClient *discovery.DiscoveryClient
	)
	job = &Job{}

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
	job.kubeconfig = kubeconfig
	job.namespace = namespace

	job.ctx = ctx
	job.config = config
	job.restClient = restClient
	job.clientset = clientset
	job.dynamicClient = dynamicClient
	job.discoveryClient = discoveryClient

	job.Options = &HandlerOptions{}

	return
}
func (j *Job) Namespace() string {
	return j.namespace
}
func (in *Job) DeepCopy() *Job {
	out := new(Job)

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
	out.SetPropagationPolicy("background")

	return out
}
func (j *Job) setNamespace(namespace string) {
	j.Lock()
	defer j.Unlock()
	j.namespace = namespace
}
func (j *Job) WithNamespace(namespace string) *Job {
	job := j.DeepCopy()
	job.setNamespace(namespace)
	return job
}
func (j *Job) WithDryRun() *Job {
	job := j.DeepCopy()
	job.Options.CreateOptions.DryRun = []string{metav1.DryRunAll}
	job.Options.UpdateOptions.DryRun = []string{metav1.DryRunAll}
	job.Options.DeleteOptions.DryRun = []string{metav1.DryRunAll}
	job.Options.PatchOptions.DryRun = []string{metav1.DryRunAll}
	job.Options.ApplyOptions.DryRun = []string{metav1.DryRunAll}
	job.SetPropagationPolicy("background")
	return job
}
func (j *Job) SetTimeout(timeout int64) {
	j.Lock()
	defer j.Unlock()
	j.Options.ListOptions.TimeoutSeconds = &timeout
}
func (j *Job) SetLimit(limit int64) {
	j.Lock()
	defer j.Unlock()
	j.Options.ListOptions.Limit = limit
}
func (j *Job) SetForceDelete(force bool) {
	j.Lock()
	defer j.Unlock()
	if force {
		gracePeriodSeconds := int64(0)
		j.Options.DeleteOptions.GracePeriodSeconds = &gracePeriodSeconds
		propagationPolicy := metav1.DeletePropagationBackground
		j.Options.DeleteOptions.PropagationPolicy = &propagationPolicy
	} else {
		j.Options.DeleteOptions = metav1.DeleteOptions{}
		propagationPolicy := metav1.DeletePropagationBackground
		j.Options.DeleteOptions.PropagationPolicy = &propagationPolicy
	}
}

// Whether and how garbage collection will be performed.
// support value are "Background", "Orphan", "Foreground",
// default value is "Background"
func (j *Job) SetPropagationPolicy(policy string) {
	j.Lock()
	defer j.Unlock()
	switch strings.ToLower(policy) {
	case strings.ToLower(string(metav1.DeletePropagationBackground)):
		propagationPolicy := metav1.DeletePropagationBackground
		j.Options.DeleteOptions.PropagationPolicy = &propagationPolicy
	case strings.ToLower(string(metav1.DeletePropagationForeground)):
		propagationPolicy := metav1.DeletePropagationForeground
		j.Options.DeleteOptions.PropagationPolicy = &propagationPolicy
	case strings.ToLower(string(metav1.DeletePropagationOrphan)):
		propagationPolicy := metav1.DeletePropagationOrphan
		j.Options.DeleteOptions.PropagationPolicy = &propagationPolicy
	default:
		propagationPolicy := metav1.DeletePropagationBackground
		j.Options.DeleteOptions.PropagationPolicy = &propagationPolicy
	}
}

// create job from bytes
func (j *Job) CreateFromBytes(data []byte) (*batchv1.Job, error) {
	jobJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	job := &batchv1.Job{}
	err = json.Unmarshal(jobJson, job)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(job.Namespace) != 0 {
		namespace = job.Namespace
	} else {
		namespace = j.namespace
	}

	return j.clientset.BatchV1().Jobs(namespace).Create(j.ctx, job, j.Options.CreateOptions)
}

// create job from file
func (j *Job) CreateFromFile(path string) (*batchv1.Job, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return j.CreateFromBytes(data)
}

// create job from file, alias to "CreateFromFile"
func (j *Job) Create(path string) (*batchv1.Job, error) {
	return j.CreateFromFile(path)
}

// update job from bytes
func (j *Job) UpdateFromBytes(data []byte) (*batchv1.Job, error) {
	jobJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	job := &batchv1.Job{}
	err = json.Unmarshal(jobJson, job)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(job.Namespace) != 0 {
		namespace = job.Namespace
	} else {
		namespace = j.namespace
	}

	return j.clientset.BatchV1().Jobs(namespace).Update(j.ctx, job, j.Options.UpdateOptions)
}

// update job from file
func (j *Job) UpdateFromFile(path string) (*batchv1.Job, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return j.UpdateFromBytes(data)
}

// update job from file, alias to "UpdateFromFile"
func (j *Job) Update(path string) (*batchv1.Job, error) {
	return j.UpdateFromFile(path)
}

// apply job from bytes
func (j *Job) ApplyFromBytes(data []byte) (job *batchv1.Job, err error) {
	job, err = j.CreateFromBytes(data)
	if errors.IsAlreadyExists(err) {
		job, err = j.UpdateFromBytes(data)
	}
	return
}

// apply job from file
func (j *Job) ApplyFromFile(path string) (job *batchv1.Job, err error) {
	job, err = j.CreateFromFile(path)
	if errors.IsAlreadyExists(err) {
		job, err = j.UpdateFromFile(path)
	}
	return
}

// apply job from file, alias to "ApplyFromFile"
func (j *Job) Apply(path string) (*batchv1.Job, error) {
	return j.ApplyFromFile(path)
}

// delete job from bytes
func (j *Job) DeleteFromBytes(data []byte) error {
	jobJson, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}

	job := &batchv1.Job{}
	err = json.Unmarshal(jobJson, job)
	if err != nil {
		return err
	}

	var namespace string
	if len(job.Namespace) != 0 {
		namespace = job.Namespace
	} else {
		namespace = j.namespace
	}

	return j.WithNamespace(namespace).DeleteByName(job.Name)
}

// delete job from file
func (j *Job) DeleteFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return j.DeleteFromBytes(data)
}

// delete job by name
func (j *Job) DeleteByName(name string) error {
	return j.clientset.BatchV1().Jobs(j.namespace).Delete(j.ctx, name, j.Options.DeleteOptions)
}

// delete job by name,alias to "DeleteByName"
func (j *Job) Delete(name string) error {
	return j.DeleteByName(name)
}

// get job from bytes
func (j *Job) GetFromBytes(data []byte) (*batchv1.Job, error) {
	jobJson, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	job := &batchv1.Job{}
	err = json.Unmarshal(jobJson, job)
	if err != nil {
		return nil, err
	}

	var namespace string
	if len(job.Namespace) != 0 {
		namespace = job.Namespace
	} else {
		namespace = j.namespace
	}

	return j.WithNamespace(namespace).GetByName(job.Name)
}

// get job from file
func (j *Job) GetFromFile(path string) (*batchv1.Job, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return j.GetFromBytes(data)
}

// get job by name
func (j *Job) GetByName(name string) (*batchv1.Job, error) {
	return j.clientset.BatchV1().Jobs(j.namespace).Get(j.ctx, name, j.Options.GetOptions)
}

// get job by name, alias to "GetByName"
func (j *Job) Get(name string) (*batchv1.Job, error) {
	return j.GetByName(name)
}

// ListByLabel list jobs by labels
func (j *Job) ListByLabel(labels string) (*batchv1.JobList, error) {
	listOptions := j.Options.ListOptions.DeepCopy()
	listOptions.LabelSelector = labels
	return j.clientset.BatchV1().Jobs(j.namespace).List(j.ctx, *listOptions)
}

// List list jobs by labels, alias to "ListByLabel"
func (j *Job) List(labels string) (*batchv1.JobList, error) {
	return j.ListByLabel(labels)
}

// ListByNamespace list jobs by namespace
func (j *Job) ListByNamespace(namespace string) (*batchv1.JobList, error) {
	return j.WithNamespace(namespace).ListByLabel("")
}

// ListAll list all jobs in the k8s cluster
func (j *Job) ListAll(namespace string) (*batchv1.JobList, error) {
	return j.WithNamespace(metav1.NamespaceAll).ListByLabel("")
}

// GetController returns a JobController object by job name if the controllee(job) has a controller.
func (j *Job) GetController(name string) (*JobController, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("not set the job name")
	}
	job, err := j.Get(name)
	if err != nil {
		return nil, err
	}
	ownerRef := metav1.GetControllerOf(job)
	if ownerRef == nil {
		return nil, fmt.Errorf("the job %q doesn't have controller", name)
	}
	oc := JobController{OwnerReference: *ownerRef}

	// new a cronjob handler
	cronjobHandler, err := NewCronJob(j.ctx, j.namespace, j.kubeconfig)
	if err != nil {
		return nil, err
	}
	cronjob, err := cronjobHandler.Get(ownerRef.Name)
	if err != nil {
		return nil, err
	}

	oc.Labels = cronjob.Labels
	oc.CreationTimestamp = cronjob.ObjectMeta.CreationTimestamp
	oc.LastScheduleTime = *(cronjob.Status.LastScheduleTime)
	oc.LastSuccessfulTime = *(cronjob.Status.LastSuccessfulTime)
	return &oc, nil
}

// check job if is completion
func (j *Job) IsComplete(name string) bool {
	// if job not exist, return false
	job, err := j.Get(name)
	if err != nil {
		return false
	}

	for _, cond := range job.Status.Conditions {
		if cond.Status == corev1.ConditionTrue && cond.Type == batchv1.JobComplete {
			return true
		}
	}

	return false
}

// check job if is condition is
// job finished means that the job condition is "complete" or "failed"
func (j *Job) IsFinish(name string) bool {
	// 1. job not exist, return true
	job, err := j.Get(name)
	if err != nil {
		return true
	}
	// 2. if job complete return true
	// 3. if job failed return true
	// 4. all other job condition return false
	for _, cond := range job.Status.Conditions {
		if cond.Status == corev1.ConditionTrue && cond.Type == batchv1.JobComplete {
			return true
		}
		if cond.Status == corev1.ConditionTrue && cond.Type == batchv1.JobFailed {
			return true
		}
	}
	return false
}

// wait job status to be "true"
func (j *Job) WaitFinish(name string) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
	)
	if j.IsFinish(name) {
		return
	}

	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: j.namespace})
		listOptions.TimeoutSeconds = &timeout
		watcher, err = j.clientset.BatchV1().Jobs(j.namespace).Watch(j.ctx, listOptions)
		if err != nil {
			return
		}
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Modified:
				if j.IsFinish(name) {
					watcher.Stop()
					return
				}
			case watch.Deleted:
				watcher.Stop()
				return fmt.Errorf("%s deleted", name)
			}
		}
	}
}

// wait job not exist
func (j *Job) WaitNotExist(name string) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
	)
	_, err = j.Get(name)
	if err != nil { // job not exist
		return nil
	}
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: j.namespace})
		listOptions.TimeoutSeconds = &timeout
		watcher, err = j.clientset.BatchV1().Jobs(j.namespace).Watch(j.ctx, listOptions)
		if err != nil {
			return
		}
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Deleted:
				for {
					if _, err := j.Get(name); err != nil { // job not exist
						break
					}
					time.Sleep(time.Millisecond * 500)
				}
				watcher.Stop()
				return
			}
		}
	}
}

// watch jobs by name
func (j *Job) WatchByName(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		timeout = int64(0)
		isExist bool
	)
	for {
		listOptions := metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: j.namespace})
		listOptions.TimeoutSeconds = &timeout
		if watcher, err = j.clientset.BatchV1().Jobs(j.namespace).Watch(j.ctx, listOptions); err != nil {
			return
		}
		if _, err = j.Get(name); err != nil {
			isExist = false // job not exist
		} else {
			isExist = true // job exist
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
				log.Debug("watch job: bookmark.")
			case watch.Error:
				log.Debug("watch job: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch job: reconnect to kubernetes")
	}
}

// watch jobs by labelSelector
func (j *Job) WatchByLabel(labelSelector string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	var (
		watcher watch.Interface
		jobList *batchv1.JobList
		timeout = int64(0)
		isExist bool
	)
	for {
		if watcher, err = j.clientset.BatchV1().Jobs(j.namespace).Watch(j.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout}); err != nil {
			return
		}
		if jobList, err = j.List(labelSelector); err != nil {
			return
		}
		if len(jobList.Items) == 0 {
			isExist = false // job not exist
		} else {
			isExist = true // job exist
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
				log.Debug("watch job: bookmark.")
			case watch.Error:
				log.Debug("watch job: error")
			}
		}
		// If event channel is closed, it means the server has closed the connection
		log.Debug("watch job: reconnect to kubernetes")
	}
}

// watch jobs by name, alias to "WatchByName"
func (j *Job) Watch(name string,
	addFunc, modifyFunc, deleteFunc func(x interface{}), x interface{}) (err error) {
	return j.WatchByName(name, addFunc, modifyFunc, deleteFunc, x)
}
