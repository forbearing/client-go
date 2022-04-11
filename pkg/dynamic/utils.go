package dynamic

import (
	"sync"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/watch"
)

func watchHandler(eventChannel <-chan watch.Event, mutex *sync.Mutex) {
	for {
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
}
