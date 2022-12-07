package configs

/**
* @Author: DK
* Description: 常量
 */

const (
	// KubeConfigPath kube 配置文件位置
	KubeConfigPath = "%s/.kube/configs"

	// ProjectDomain 项目域名
	ProjectDomain = "0.0.0.0"

	// ProjectPort 项目端口
	ProjectPort = "8919"

	CertFile = "./certs/server.pem"
	KeyFile  = "./certs/server-key.pem"

	Default = ""
	DefaultApiProxyPrefix = "/"

	ClusterParam = "labelSelector"
	ClusterKey   = "cluster"

	// 响应做修改时的正则
	Myres_pattern             = `/apis/apps/v1` //- // mycluster 强制 归类于 apps
	MyCluster_pattern_handler = `/apis/apps/v1/namespaces/.*?/myclusters`
	MyResourceApiVersion      = "apps/v1"
	MyClusterName             = "myclusters"
	MyClusterKind             = "MyCluster"
	MyClusterListKind         = "MyClusterList"
	MyClusterShortName        = "mc"
)
