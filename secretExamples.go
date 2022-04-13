package main

import (
	"hybfkuf/pkg/k8s"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func secretExamples() {
	var (
		yamlfile      = "./testData/secret.yaml"
		name          = "test"
		labelSelector = "type=secret"
		forceDelete   = false
	)
	secret, err := k8s.NewSecret(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = secret

	// 1. create secret
	secret.Delete(name)
	if s, err := secret.Create(yamlfile); err != nil {
		log.Error("create secret failed")
		log.Error(err)
	} else {
		log.Infof("create secret %s success.", s.Name)
	}
	// 2. update secret
	if s, err := secret.Update(yamlfile); err != nil {
		log.Error("update secret failed")
		log.Error(err)
	} else {
		log.Infof("update secret %s success.", s.Name)
	}
	// 3. apply secret
	if s, err := secret.Apply(yamlfile); err != nil {
		log.Error("apply secret failed")
		log.Error(err)
	} else {
		log.Infof("apply secret %s success.", s.Name)
	}
	secret.Delete(name)
	if s, err := secret.Apply(yamlfile); err != nil {
		log.Error("apply secret failed")
		log.Error(err)
	} else {
		log.Infof("apply secret %s success.", s.Name)
	}
	// 4. delete secret
	if err := secret.Delete(name); err != nil {
		log.Error("delete secret failed")
		log.Error(err)
	} else {
		log.Infof("delete secret %s success.", name)
	}
	// 5. get secret
	secret.Create(yamlfile)
	if s, err := secret.Get(name); err != nil {
		log.Error("get secret failed")
		log.Error(err)
	} else {
		log.Infof("get secret %s success.", s.Name)
	}
	// 6. list secret
	if sl, err := secret.List(labelSelector); err != nil {
		log.Error("list secret failed")
		log.Error(err)
	} else {
		log.Info("list secret success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	// 7. watch secret
	log.Info("start watch secret")
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Second * time.Duration(rand.Intn(5)))
			secret.Apply(yamlfile)
		}
	}()
	go func() {
		for {
			rand.Seed(time.Now().UnixNano())
			//time.Sleep(time.Second * time.Duration(rand.Intn(30)))
			time.Sleep(time.Second * 10)
			secret.Delete(name)
		}
	}()
	secret.Watch(name,
		func(x interface{}) {
			log.Info("add secret.")
		},
		func(x interface{}) {
			log.Info("modified secret.")
		},
		func(x interface{}) {
			log.Info("deleted secret.")
		},
		nil,
	)
}
