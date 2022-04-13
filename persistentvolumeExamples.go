package main

import (
	"time"

	"hybfkuf/pkg/k8s"

	log "github.com/sirupsen/logrus"
)

func persistentvolumeExamples() {
	var (
		yamlfile      = "./testData/persistentvolume.yaml"
		name          = "test"
		labelSelector = "type=persistentvolume"
		forceDelete   = false
	)
	persistentvolume, err := k8s.NewPersistentVolume(ctx, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = yamlfile
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = persistentvolume

	// 1. create persistentvolume
	persistentvolume.Delete(name)
	time.Sleep(time.Second * 5)
	if s, err := persistentvolume.Create(yamlfile); err != nil {
		log.Error("create persistentvolume failed")
		log.Error(err)
	} else {
		log.Infof("create persistentvolume %s success.", s.Name)
	}
	// 2. update persistentvolume
	if s, err := persistentvolume.Update(yamlfile); err != nil {
		log.Error("update persistentvolume failed")
		log.Error(err)
	} else {
		log.Infof("update persistentvolume %s success.", s.Name)
	}
	// 3. apply persistentvolume
	if s, err := persistentvolume.Apply(yamlfile); err != nil {
		log.Error("apply persistentvolume failed")
		log.Error(err)
	} else {
		log.Infof("apply persistentvolume %s success.", s.Name)
	}
	persistentvolume.Delete(name)
	time.Sleep(time.Second * 5)
	if s, err := persistentvolume.Apply(yamlfile); err != nil {
		log.Error("apply persistentvolume failed")
		log.Error(err)
	} else {
		log.Infof("apply persistentvolume %s success.", s.Name)
	}
	// 4. delete persistentvolume
	if err := persistentvolume.Delete(name); err != nil {
		log.Error("delete persistentvolume failed")
		log.Error(err)
	} else {
		log.Infof("delete persistentvolume %s success.", name)
	}
	// 5. get persistentvolume
	time.Sleep(time.Second * 5)
	persistentvolume.Create(yamlfile)
	if s, err := persistentvolume.Get(name); err != nil {
		log.Error("get persistentvolume failed")
		log.Error(err)
	} else {
		log.Infof("get persistentvolume %s success.", s.Name)
	}
	// 6. list persistentvolume
	if sl, err := persistentvolume.List(labelSelector); err != nil {
		log.Error("list persistentvolume failed")
		log.Error(err)
	} else {
		log.Info("list persistentvolume success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	//// 7. watch persistentvolume
	//log.Info("start watch persistentvolume")
	//go func() {
	//    for {
	//        rand.Seed(time.Now().UnixNano())
	//        time.Sleep(time.Second * time.Duration(rand.Intn(5)))
	//        persistentvolume.Apply(yamlfile)
	//    }
	//}()
	//go func() {
	//    for {
	//        rand.Seed(time.Now().UnixNano())
	//        //time.Sleep(time.Second * time.Duration(rand.Intn(30)))
	//        time.Sleep(time.Second * 30)
	//        persistentvolume.Delete(name, forceDelete)
	//    }
	//}()
	//persistentvolume.Watch(name,
	//    func(x interface{}) {
	//        log.Info("add persistentvolume.")
	//    },
	//    func(x interface{}) {
	//        log.Info("modified persistentvolume.")
	//    },
	//    func(x interface{}) {
	//        log.Info("deleted persistentvolume.")
	//    },
	//    nil,
	//)

	// 8. get the pvc name of the pv
	name = "pvc-0e7e4a7a-e090-45b1-b2b1-5cdbdd597c83"
	//name = "pv001"
	pvc, err := persistentvolume.GetPVC(name)
	if err != nil {
		log.Error("get pvc name failed")
		log.Error(err)
	} else {
		log.Info("the pvc name is:")
		log.Info(pvc)
	}

	// 9. get the storageclass name of the pv
	sc, err := persistentvolume.GetStorageClass(name)
	if err != nil {
		log.Error("get storageclass failed")
		log.Error(err)
	} else {
		log.Info("the storageclass is:")
		log.Info(sc)
	}
	// 10. get the accessModes of the pv
	accessModes, err := persistentvolume.GetAccessModes(name)
	if err != nil {
		log.Error("get accessModes failed")
		log.Error(err)
	} else {
		log.Info("the accessModes is:")
		log.Info(accessModes)
	}

	// 10. get the capacity of the pv
	capacity, err := persistentvolume.GetCapacity(name)
	if err != nil {
		log.Error("get persistentvolume failed")
		log.Error(err)
	} else {
		log.Info("the capacity is:")
		log.Info(capacity)
	}

	// 11. get he status phase of the pv
	phase, err := persistentvolume.GetPhase(name)
	if err != nil {
		log.Error("get persistentvolume failed")
		log.Error(err)
	} else {
		log.Info("the status phase is:")
		log.Info(phase)
	}

	// 12. get the reclaim policy of the pv
	policy, err := persistentvolume.GetReclaimPolicy(name)
	if err != nil {
		log.Error("get persistentvolume failed")
		log.Error(err)
	} else {
		log.Info("the reclaim policy is:")
		log.Info(policy)
	}
}
