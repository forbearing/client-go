package dynamic

//var kubeconfig *string

//func init() {
//    if home := homedir.HomeDir(); home != "" {
//        kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
//    } else {
//        kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
//    }
//    flag.Parse()
//}

//// New 创建一个 dynamic 客户端
//func New() (client dynamic.Interface, error error) {
//    // use the current context in kubeconfig
//    config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
//    if err != nil {
//        return
//    }
//    // create the dynamic
//    client, err = dynamic.NewForConfig(config)
//    if err != nil {
//        return
//    }
//    return
//}
