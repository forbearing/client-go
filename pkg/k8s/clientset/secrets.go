package clientset

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Secret struct {
	namespace string
	limit     int64
	timeout   int64
	ctx       context.Context
	client    *kubernetes.Clientset

	sync.Mutex
}

// new a `Secret` instance from kubeconfig or in-cluster config
func NewSecret(ctx context.Context, namespace, kubeconfig string) (secret *Secret, err error) {
	var (
		config *rest.Config
		client *kubernetes.Clientset
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
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return
		}
	}

	// create the clientset
	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	secret.namespace = namespace
	secret.limit = 100
	secret.timeout = 10
	secret.ctx = ctx
	secret.client = client

	return
}
func (s *Secret) SetLimit(limit int64) {
	s.Lock()
	defer s.Unlock()
	s.limit = limit
}
func (s *Secret) SetTimeout(timeout int64) {
	s.Lock()
	defer s.Unlock()
	s.timeout = timeout
}

// create secret from file
func (s *Secret) Create(filepath string) (secret *corev1.Secret, err error) {
	var (
		secretYaml []byte
		secretJson []byte
	)
	secret = &corev1.Secret{}
	if secretYaml, err = ioutil.ReadFile(filepath); err != nil {
		return
	}
	if secretJson, err = yaml.ToJSON(secretYaml); err != nil {
		return
	}
	if err = json.Unmarshal(secretJson, secret); err != nil {
		return
	}
	secret, err = s.client.CoreV1().Secrets(s.namespace).Create(s.ctx, secret, metav1.CreateOptions{})
	return
}

// update secret from file
func (s *Secret) Update(filepath string) (secret *corev1.Secret, err error) {
	var (
		secretYaml []byte
		secretJson []byte
	)
	secret = &corev1.Secret{}
	if secretYaml, err = ioutil.ReadFile(filepath); err != nil {
		return
	}
	if secretJson, err = yaml.ToJSON(secretYaml); err != nil {
		return
	}
	if err = json.Unmarshal(secretJson, secret); err != nil {
		return
	}
	secret, err = s.client.CoreV1().Secrets(s.namespace).Update(s.ctx, secret, metav1.UpdateOptions{})
	return
}

// apply secret from file
func (s *Secret) Apply(filepath string) (secret *corev1.Secret, err error) {
	secret, err = s.Create(filepath)
	if errors.IsAlreadyExists(err) {
		secret, err = s.Update(filepath)
	}
	return
}

// delete secret by name
func (s *Secret) Delete(name string, forceDelete bool) (err error) {
	var gracePeriodSeconds int64
	if forceDelete {
		gracePeriodSeconds = 0
		err = s.client.CoreV1().Secrets(s.namespace).Delete(s.ctx,
			name, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds})
		return
	}
	err = s.client.CoreV1().Secrets(s.namespace).Delete(s.ctx, name, metav1.DeleteOptions{})
	return
}

// get secret by name
func (s *Secret) Get(name string) (secret *corev1.Secret, err error) {
	secret, err = s.client.CoreV1().Secrets(s.namespace).Get(s.ctx, name, metav1.GetOptions{})
	return
}

// list secret by labelSelector
func (s *Secret) List(labelSelector string) (secretList *corev1.SecretList, err error) {
	secretList, err = s.client.CoreV1().Secrets(s.namespace).List(s.ctx,
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &s.timeout, Limit: s.limit})
	return
}

// watch secret by labelSelector
func (c *Secret) Watch(labelSelector string, addFunc, modifyFunc, deleteFunc func()) (err error) {
	var (
		watcher    watch.Interface
		secretList *corev1.SecretList
		timeout    = int64(0)
		isExist    bool
	)
	for {
		if watcher, err = c.client.CoreV1().Secrets(c.namespace).Watch(c.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout, Limit: c.limit}); err != nil {
			logrus.Error(err)
			return
		}
		if secretList, err = c.List(labelSelector); err != nil {
			logrus.Error(err)
			return
		}
		if len(secretList.Items) == 0 {
			isExist = false // secret not exist
		} else {
			isExist = true // secret exist
		}
		for {
			event, isOpen := <-watcher.ResultChan()
			if isOpen {
				switch event.Type {
				case watch.Added:
					if !isExist {
						addFunc()
					}
					isExist = true
				case watch.Modified:
					modifyFunc()
					isExist = true
				case watch.Deleted:
					deleteFunc()
					isExist = false
				default: // no nothing
				}
			} else {
				// If event channel is closed, it means the server has closed the connection
				logrus.Info("reconnect to kube-apiserver")
				break
			}
		}
	}
}
