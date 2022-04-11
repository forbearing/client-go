package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	seryaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	filePath string
)

//func init() {
//    flag.StringVar(&filePath, "path", "./examples/deployment.yaml", "yaml file path")
//    flag.Parse()
//}

func applyExamples() {
	//path := "./examples/deployment.yaml"
	path := "./examples/all.yaml"
	_, err := apply(path)
	if err != nil {
		log.Error("apply failed")
		log.Error(err)
	} else {
		log.Info("apply success.")
	}
}

func apply(path string) (*appsv1.Deployment, error) {
	kubeconfig := "/Users/hybfkuf/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	yamlData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	jsonData, err := utilyaml.ToJSON(yamlData)
	if err != nil {
		return nil, err
	}
	deploy := &appsv1.Deployment{}
	err = json.Unmarshal(jsonData, deploy)
	if err != nil {
		return nil, err
	}

	bufferSize := 500
	decoder := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlData), bufferSize)
	for {
		var rawObject runtime.RawExtension
		if err := decoder.Decode(&rawObject); err != nil {
			break
		}
		if len(rawObject.Raw) == 0 {
			// if the yaml object is empty just continue to the next one
			continue
		}
		object, gvk, err := seryaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObject.Raw, nil, nil)
		if err != nil {
			return nil, err
		}
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
		if err != nil {
			return nil, err
		}
		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
		if err != nil {
			return nil, err
		}
		restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)
		restMapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)

		var dri dynamic.ResourceInterface
		if restMapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace("default")
			}
			dri = dynamicClient.Resource(restMapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = dynamicClient.Resource(restMapping.Resource)
		}

		log.Info("create k8s resource")
		_, err = dri.Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
		if k8serrors.IsAlreadyExists(err) {
			log.Info("update k8s resource")
			_, err = dri.Update(context.TODO(), unstructuredObj, metav1.UpdateOptions{})
		}
		if err != nil {
			return nil, err
		}

	}

	return nil, nil
}
