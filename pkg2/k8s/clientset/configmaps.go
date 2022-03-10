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

type ConfigMap struct {
	namespace string
	limit     int64
	timeout   int64
	ctx       context.Context
	client    *kubernetes.Clientset

	sync.Mutex
}

// new a `ConfigMap` instance from kubeconfig or in-cluster config
func NewConfigMap(ctx context.Context, namespace, kubeconfig string) (configMap *ConfigMap, err error) {
	var (
		config *rest.Config
		client *kubernetes.Clientset
	)
	configMap = &ConfigMap{}

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

	configMap.namespace = namespace
	configMap.limit = 100
	configMap.timeout = 10
	configMap.ctx = ctx
	configMap.client = client

	return
}
func (c *ConfigMap) SetLimit(limit int64) {
	c.Lock()
	defer c.Unlock()
	c.limit = limit
}
func (c *ConfigMap) SetTimeout(timeout int64) {
	c.Lock()
	defer c.Unlock()
	c.timeout = timeout
}

// create configMap from file
func (c *ConfigMap) Create(filepath string) (configMap *corev1.ConfigMap, err error) {
	var (
		cmYaml []byte
		cmJson []byte
	)
	configMap = &corev1.ConfigMap{}
	if cmYaml, err = ioutil.ReadFile(filepath); err != nil {
		return
	}
	if cmJson, err = yaml.ToJSON(cmYaml); err != nil {
		return
	}
	if err = json.Unmarshal(cmJson, configMap); err != nil {
		return
	}
	configMap, err = c.client.CoreV1().ConfigMaps(c.namespace).Create(c.ctx, configMap, metav1.CreateOptions{})
	return
}

// update configMap from file
func (c *ConfigMap) Update(filepath string) (configMap *corev1.ConfigMap, err error) {
	var (
		cmYaml []byte
		cmJson []byte
	)
	configMap = &corev1.ConfigMap{}
	if cmYaml, err = ioutil.ReadFile(filepath); err != nil {
		return
	}
	if cmJson, err = yaml.ToJSON(cmYaml); err != nil {
		return
	}
	if err = json.Unmarshal(cmJson, configMap); err != nil {
		return
	}
	configMap, err = c.client.CoreV1().ConfigMaps(c.namespace).Update(c.ctx, configMap, metav1.UpdateOptions{})
	return
}

// apply configMap from file
func (c *ConfigMap) Apply(filepath string) (configMap *corev1.ConfigMap, err error) {
	configMap, err = c.Create(filepath)
	if errors.IsAlreadyExists(err) {
		configMap, err = c.Update(filepath)
	}
	return
}

// delete configMap by name
func (c *ConfigMap) Delete(name string, forceDelete bool) (err error) {
	var gracePeriodSeconds int64
	if forceDelete {
		gracePeriodSeconds = 0
		err = c.client.CoreV1().ConfigMaps(c.namespace).Delete(c.ctx,
			name, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds})
		return
	}
	err = c.client.CoreV1().ConfigMaps(c.namespace).Delete(c.ctx, name, metav1.DeleteOptions{})
	return
}

// get configMap by name
func (c *ConfigMap) Get(name string) (configMap *corev1.ConfigMap, err error) {
	configMap, err = c.client.CoreV1().ConfigMaps(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
	return
}

// list configMap by labelSelector
func (c *ConfigMap) List(labelSelector string) (configMapList *corev1.ConfigMapList, err error) {
	configMapList, err = c.client.CoreV1().ConfigMaps(c.namespace).List(c.ctx,
		metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &c.timeout, Limit: c.limit})
	return
}

// watch configMap by labelSelector
func (c *ConfigMap) Watch(labelSelector string, addFunc, modifyFunc, deleteFunc func()) (err error) {
	var (
		watcher       watch.Interface
		configMapList *corev1.ConfigMapList
		timeout       = int64(0)
		isExist       bool
	)
	for {
		if watcher, err = c.client.CoreV1().ConfigMaps(c.namespace).Watch(c.ctx,
			metav1.ListOptions{LabelSelector: labelSelector, TimeoutSeconds: &timeout, Limit: c.limit}); err != nil {
			logrus.Error(err)
			return
		}
		if configMapList, err = c.List(labelSelector); err != nil {
			logrus.Error(err)
			return
		}
		if len(configMapList.Items) == 0 {
			isExist = false // configMap not exist
		} else {
			isExist = true // configMap exist
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
