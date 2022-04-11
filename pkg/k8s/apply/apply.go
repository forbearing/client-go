//package apply

//import (
//    "bytes"
//    "context"
//    "fmt"
//    "io"

//    corev1 "k8s.io/api/core/v1"
//    "k8s.io/apimachinery/pkg/api/meta"
//    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
//    "k8s.io/apimachinery/pkg/runtime"
//    "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
//    "k8s.io/apimachinery/pkg/types"
//    k8syaml "k8s.io/apimachinery/pkg/util/yaml"
//    "k8s.io/client-go/discovery"
//    "k8s.io/client-go/dynamic"
//    "k8s.io/client-go/restmapper"
//    "k8s.io/klog"
//    "k8s.io/kubectl/pkg/scheme"
//)

//const DefaultDecoderBufferSize = 500

//type applyOptions struct {
//    dynamicClient   dynamic.Interface
//    discoveryClient discovery.DiscoveryInterface
//    serverSide      bool
//}

//func NewApplyOptions(dynamicClient dynamic.Interface, discoveryClient discovery.DiscoveryInterface) *applyOptions {
//    return &applyOptions{
//        dynamicClient:   dynamicClient,
//        discoveryClient: discoveryClient,
//    }
//}
//func (o *applyOptions) WithServerSide(serverSide bool) *applyOptions {
//    o.serverSide = serverSide
//    return o
//}

//// discovery.DiscoveryInterface -> []*APIGroupResources -> meta.RESTMapper
//func (o *applyOptions) ToRESTMapper() (meta.RESTMapper, error) {
//    gv, err := restmapper.GetAPIGroupResources(o.discoveryClient)
//    if err != nil {
//        return nil, err
//    }
//    restMapper := restmapper.NewDiscoveryRESTMapper(gv)
//    return restMapper, nil
//}

//// data []byte + RESTMapper -> apply yaml file
//func (o *applyOptions) Apply(ctx context.Context, data []byte) error {
//    restMapper, err := o.ToRESTMapper()
//    if err != nil {
//        return err
//    }
//    unstructList, err := Decode(data)
//    if err != nil {
//        return err
//    }

//    for _, unstruct := range unstructList {
//        klog.V(5).Infof("Apply object: %#v", unstruct)
//        j
//    }
//    return nil
//}

//// data []byte -> []unstructured.Unstructured
//// []byte -> runtime.Object -> map[string]interface{} -> unstructured.Unstructured -> []unstructured.Unstructured
//func Decode(data []byte) (unstructList []unstructured.Unstructured, err error) {
//    decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), DefaultDecoderBufferSize)

//    i := 1
//    for {
//        var reqObj runtime.RawExtension
//        if err = decoder.Decode(&reqObj); err != nil {
//            break
//        }
//        klog.V(5).Infof("The section:[%d] raw content: %s", i, string(reqObj.Raw))
//        obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(reqObj.Raw, nil, nil)
//        if err != nil {
//            err = fmt.Errorf("serialize the section:[%d] content error, %v", i, err)
//            break
//        }
//        klog.V(5).Infof("The section:[%d] GroupVersionKind: %#v  object: %#v", i, gvk, obj)

//        unstruct, err := ConvertSingleObjectToUnstructured(obj)
//        if err != nil {
//            err = fmt.Errorf("serialize the section:[%d] content error, %v", i, err)
//            break
//        }
//        unstructList = append(unstructList, unstruct)
//        i++
//    }
//    if err != io.EOF {
//        return unstructList, fmt.Errorf("parsing the section:[%d] content error, %v", i, err)
//    }

//    return unstructList, nil
//}

//// []byte -> runtime.Object
//// runtime.Object 可以转换成 unstructured.Unstructured 或 *appsv1.Deployment
//// runtime.Object 转换成 unstructured.Unstructured 使用 dynamic 客户端来操作 k8s
//// runtime.Object 转换成 appsv1.Deployment 等使用 clientset 客户端来操作 k8s
//func Decode2(data []byte) (object runtime.Object, err error) {
//    decode := scheme.Codecs.UniversalDeserializer().Decode
//    object, _, err = decode(data, nil, nil)
//    return
//}

//func ApplyUnstructured(ctx context.Context,
//    dynamicClient dynamic.Interface, restMapper meta.RESTMapper,
//    unstructuredObj unstructured.Unstructured, serverSide bool) (*unstructured.Unstructured, error) {
//    if len(unstructuredObj.GetName()) == 0 {
//        metadata, _ := meta.Accessor(unstructuredObj)
//        generateName := metadata.GetGenerateName()
//        if len(generateName) > 0 {
//            return nil, fmt.Errorf("from %s: cannot use generate name with apply", generateName)
//        }
//    }
//    // unstructuredObj -> JSON []byte
//    b, err := unstructuredObj.MarshalJSON()
//    if err != nil {
//        return nil, err
//    }
//    // JSON []byte ->  runtime.Object
//    obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(b, nil, nil)
//    if err != nil {
//        return nil, err
//    }

//    mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
//    if err != nil {
//        return nil, err
//    }
//    klog.V(5).Infof("mapping: %v", mapping.Scope.Name())
//    var dir dynamic.ResourceInterface
//    if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
//        if unstructuredObj.GetNamespace() == "" {
//            unstructuredObj.SetNamespace("default")
//        }
//        dri = dynamicClient.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
//    } else {
//        dri = dynamicClient.Resource(mapping.Resource)
//    }
//    if serverSide {
//        klog.V(2).Infof("Using server-side apply")
//        if _, ok := unstructuredObj.GetAnnotations()[corev1.LastAppliedConfigAnnotation]; ok {
//            annotations := unstructuredObj.GetAnnotations()
//            delete(annotations, corev1.LastAppliedConfigAnnotation)
//            unstructuredObj.SetAnnotations(annotations)
//        }
//        unstructuredObj.SetManagedFields(nil)
//        klog.V(4).Infof("Need remove managedFields before apply, %#v", unstructuredObj)

//        force := true
//        opts := metav1.PatchOptions{FieldManager: "k8sutil", Force: &force}
//        if _, err := dri.Patch(ctx, unstructuredObj.GetName(), types.ApplyPatchType, b, opts); err != nil {
//            if isIncompatibleServerError(err) {
//                err = fmt.Errorf("server-side apply not available on the server: (%v)", err)
//            }
//            return nil, err
//        }
//        return nil, err
//    }

//    return nil, nil
//}

//func ConvertObjectToUnstructuredList(obj runtime.Object) ([]unstructured.Unstructured, error) {
//    list := make([]unstructured.Unstructured, 0, 0)
//    if meta.IsListType(obj) {
//        if _, ok := obj.(*unstructured.UnstructuredList); !ok {
//            return nil, fmt.Errorf("unable to convert runtime object to list")
//        }
//        for _, u := range obj.(*unstructured.UnstructuredList).Items {
//            list = append(list, u)
//        }
//        return list, nil
//    }
//    unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
//    if err != nil {
//        return nil, err
//    }
//    unstructuredObj := unstructured.Unstructured{Object: unstructuredMap}
//    list = append(list, unstructuredObj)
//    return list, nil
//}
//func ConvertSingleObjectToUnstructured(object runtime.Object) (unstructured.Unstructured, error) {
//    unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
//    if err != nil {
//        return unstructured.Unstructured{}, err
//    }
//    return unstructured.Unstructured{Object: unstructuredMap}, nil
//}
