package configs

import (
	"fmt"
	"io/ioutil"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"log"
	"os"
)

/**
* @Author: DK
* @Date: 2022/11/29 16:20
* Description: 描述
* Updated:时间@版本@变更说明
 */
const (
	// 这是一个写死的包含 多集群的配置文件 路径 ---- 要改自己改
	kubeConfigFilePath = "D:\\coding\\GO\\kubenetes-study\\kubectl-proxy-m\\resources\\config"
)


type ClusterService struct {
	ApiConfig *api.Config
}

func NewClusterService() *ClusterService {
	return &ClusterService{ApiConfig: KubeApiConfig()}
}



type RestConfig struct {
	RestCfg *rest.Config
	IsDefault  bool //是否是默认的
}



// 读取 配置文件
func KubeApiConfig() *api.Config {
	configFile, err := os.Open(kubeConfigFilePath)
	if err != nil {
		log.Fatal("configFile",err)
	}
	b, _ := ioutil.ReadAll(configFile)
	cc, err := clientcmd.NewClientConfigFromBytes(b)
	if err != nil {
		log.Fatal("cc-",err)
	}
	apiConfig, err := cc.RawConfig()

	if err != nil {
		log.Fatal("apiConfig",err)
	}
	return &apiConfig
}

// 获取 Context--RestConfig 的对应map
// key是context名称， value是RestConfig 对象 方便外部调用
func (this *ClusterService) GetContextRestConfigMap() map[string]*RestConfig {
//func (this *ClusterService) GetContextRestConfigMap() map[string]*rest.Config {
//	ret := make(map[string]*rest.Config)
	ret := make(map[string]*RestConfig)

	// 遍历context
	for ctxName, _ := range this.ApiConfig.Contexts {
		restConfig, err := this.GetRestConfigByContextName(ctxName)
		if err != nil {
			continue //代表加载有错，不做处理
		}
		//ret[ctxName] = &RestConfig{RestCfg: restConfig}
		ret[ctxName] = &RestConfig{RestCfg: restConfig,
			IsDefault: this.ApiConfig.CurrentContext == ctxName}

		//ret[ctxName] = restConfig
	}
	return ret

}

func (this *ClusterService) GetRestConfigByContextName(ctxName string) (*rest.Config, error) {
	joinClusterConfig, err := clientcmd.BuildConfigFromKubeconfigGetter("", func() (*api.Config, error) {
		copyConfig := this.ApiConfig.DeepCopy()
		copyConfig.CurrentContext = ctxName
		return copyConfig, nil
	})
	if err != nil {
		return nil, err
	}
	return joinClusterConfig, nil
}


//根据 clustername获取 对应的 rest.Config对象
func (this *ClusterService) GetRestConfigByClusterName(clusterName string) (*rest.Config, error) {
	joinClusterCtx := this.GetContextByCluterName(clusterName)
	if joinClusterCtx == "" {
		return nil, fmt.Errorf("found not cluster-context by cluster-name %s", clusterName)
	}
	return this.GetRestConfigByContextName(joinClusterCtx)
}

//根据name找到cluster信息
func (this *ClusterService) GetClusterInfoByName(name string) (*api.Cluster, error) {

	for clusterName, clusterValue := range this.ApiConfig.Clusters {
		if clusterName == name { //这里取到context .
			return clusterValue, nil
		}
	}
	return nil, fmt.Errorf("cluster not found")
}

//根据集群名称 获取 context名称
func (this *ClusterService) GetContextByCluterName(name string) string {
	for ctxName, ctxValue := range this.ApiConfig.Contexts {
		if ctxValue.Cluster == name { //匹配到cluster
			return ctxName //直接返回  context名称
		}
	}
	return ""
}
