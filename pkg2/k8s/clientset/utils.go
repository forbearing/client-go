package clientset

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
)

func watchHandler(eventChannel <-chan watch.Event, mutex *sync.Mutex) {
	for { // {{{
		event, open := <-eventChannel
		if open {
			switch event.Type {
			case watch.Added:
				logrus.Info("add")
			case watch.Modified:
				logrus.Info("modified")
			case watch.Deleted:
				logrus.Info("deleted")
			case watch.Bookmark:
				logrus.Info("bookmark")
			case watch.Error:
				logrus.Error("error")
			default: // do nothing
			}
		} else {
			// If eventChannel is closed, it means the server has closed the connection
			return
		}
	}
} // }}}
func createDeployment(clientset *kubernetes.Clientset) {
	var ( // {{{
		NAMESPACE  = "default"
		DEPLOYMENT = "nginx"
	)
	// 得到 deployment 客户端
	deploymentClient := clientset.AppsV1().Deployments(NAMESPACE)
	// 实例化一个数据结构
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: DEPLOYMENT},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(2),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "nginx"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "nginx"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "nginx",
						Image:           "nginx",
						ImagePullPolicy: "IfNotPresent",
						Ports: []corev1.ContainerPort{{
							Name:          "http",
							Protocol:      corev1.ProtocolTCP,
							ContainerPort: 80,
						}},
					}},
				},
			},
		},
	}

	result, err := deploymentClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Create deployment: %s\n", result.GetName())
} // }}}
