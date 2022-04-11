package main

import (
	"hybfkuf/pkg/k8s"
	"time"

	log "github.com/sirupsen/logrus"
)

func persistentvolumeclaimExamples() {
	var (
		filepath      = "./examples/persistentvolumeclaim.yaml"
		name          = "test"
		labelSelector = "type=persistentvolumeclaim"
		forceDelete   = false
	)
	persistentvolumeclaim, err := k8s.NewPersistentVolumeClaim(ctx, NAMESPACE, *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	_ = filepath
	_ = name
	_ = labelSelector
	_ = forceDelete
	_ = persistentvolumeclaim

	// 1. create persistentvolumeclaim
	persistentvolumeclaim.Delete(name)
	if s, err := persistentvolumeclaim.Create(filepath); err != nil {
		log.Error("create persistentvolumeclaim failed")
		log.Error(err)
	} else {
		log.Infof("create persistentvolumeclaim %s success.", s.Name)
	}
	// 2. update persistentvolumeclaim
	if s, err := persistentvolumeclaim.Update(filepath); err != nil {
		log.Error("update persistentvolumeclaim failed")
		log.Error(err)
	} else {
		log.Infof("update persistentvolumeclaim %s success.", s.Name)
	}
	// 3. apply persistentvolumeclaim
	if s, err := persistentvolumeclaim.Apply(filepath); err != nil {
		log.Error("apply persistentvolumeclaim failed")
		log.Error(err)
	} else {
		log.Infof("apply persistentvolumeclaim %s success.", s.Name)
	}
	persistentvolumeclaim.Delete(name)
	if s, err := persistentvolumeclaim.Apply(filepath); err != nil {
		log.Error("apply persistentvolumeclaim failed")
		log.Error(err)
	} else {
		log.Infof("apply persistentvolumeclaim %s success.", s.Name)
	}
	// 4. delete persistentvolumeclaim
	if err := persistentvolumeclaim.Delete(name); err != nil {
		log.Error("delete persistentvolumeclaim failed")
		log.Error(err)
	} else {
		log.Infof("delete persistentvolumeclaim %s success.", name)
	}
	// 5. get persistentvolumeclaim
	time.Sleep(time.Second * 3)
	persistentvolumeclaim.Create(filepath)
	time.Sleep(time.Second * 2)
	if s, err := persistentvolumeclaim.Get(name); err != nil {
		log.Error("get persistentvolumeclaim failed")
		log.Error(err)
	} else {
		log.Infof("get persistentvolumeclaim %s success.", s.Name)
	}
	// 6. list persistentvolumeclaim
	if sl, err := persistentvolumeclaim.List(labelSelector); err != nil {
		log.Error("list persistentvolumeclaim failed")
		log.Error(err)
	} else {
		log.Info("list persistentvolumeclaim success.")
		for _, s := range sl.Items {
			log.Info(s.Name)
		}
	}
	//// 7. watch persistentvolumeclaim
	//log.Info("start watch persistentvolumeclaim")
	//go func() {
	//    for {
	//        rand.Seed(time.Now().UnixNano())
	//        time.Sleep(time.Second * time.Duration(rand.Intn(5)))
	//        persistentvolumeclaim.Apply(filepath)
	//    }
	//}()
	//go func() {
	//    for {
	//        rand.Seed(time.Now().UnixNano())
	//        //time.Sleep(time.Second * time.Duration(rand.Intn(30)))
	//        time.Sleep(time.Second * 10)
	//        persistentvolumeclaim.Delete(name, forceDelete)
	//    }
	//}()
	//persistentvolumeclaim.Watch(name,
	//    func(x interface{}) {
	//        log.Info("add persistentvolumeclaim.")
	//    },
	//    func(x interface{}) {
	//        log.Info("modified persistentvolumeclaim.")
	//    },
	//    func(x interface{}) {
	//        log.Info("deleted persistentvolumeclaim.")
	//    },
	//    nil,
	//)

	// 8 get the pv name of the persistentvolumeclaim
	pv, err := persistentvolumeclaim.GetPV(name)
	if err != nil {
		log.Error("get the pv name faild")
		log.Error(err)
	} else {
		log.Info("the pv is:")
		log.Info(pv)
	}
	// 9. get the storageclas name of the persistentvolumeclaim
	sc, err := persistentvolumeclaim.GetStorageClass(name)
	if err != nil {
		log.Error("get storageclas failed")
		log.Error(err)
	} else {
		log.Info("the storageclass is:")
		log.Info(sc)
	}
	// 10. get the accessModes of the persistentvolumeclaim
	accessModes, err := persistentvolumeclaim.GetAccessModes(name)
	if err != nil {
		log.Error("get access modes failed")
		log.Error(err)
	} else {
		log.Info("the access modes is:")
		log.Info(accessModes)
	}
	// 11. get the capacity of the persistentvolumeclaim
	capacity, err := persistentvolumeclaim.GetCapacity(name)
	if err != nil {
		log.Error("get capacity failed")
		log.Error(err)
	} else {
		log.Println("the capacity is:")
		log.Info(capacity)
	}
	// 12. Get the status phase of the persistentvolumeclaim
	phase, err := persistentvolumeclaim.GetPhase(name)
	if err != nil {
		log.Error("get phase failed")
		log.Error(err)
	} else {
		log.Println("the status phase is:")
		log.Info(phase)
	}
}
