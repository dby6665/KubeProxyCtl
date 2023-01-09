package configs

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
)

/**
* @Author: DK
* Description: 读取 k8s configs 文件 并初始化
 */

//全局变量

type K8sConfig struct{}

//直接初始化
func NewK8sConfig() *K8sConfig {
	cfg := &K8sConfig{}

	return cfg
}

//从 ~/.kube/config取的
func (*K8sConfig) K8sRestConfigDefault() *rest.Config {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	defaultConfigPath := fmt.Sprintf(KubeConfigPath, home)
	fmt.Println(defaultConfigPath)

	config, err := clientcmd.BuildConfigFromFlags("", "D:\\coding\\GO\\kubeProxyCtl\\tools\\utils\\configs\\config")
	if err != nil {
		log.Fatal(err)
	}
	return config
}

//初始化client-go客户端
func (cfg *K8sConfig) InitClient(restC *rest.Config) *kubernetes.Clientset {
	c, err := kubernetes.NewForConfig(restC)
	if err != nil {
		log.Fatal(err)
	}

	return c
}
