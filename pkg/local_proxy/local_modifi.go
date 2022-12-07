package local_proxy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"KubeProxyCtl/pkg/helper"
	"KubeProxyCtl/tools/utils/configs"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/**
* @Author: DK
* Description: 修改内容
 */

type MyHttpHandler struct {
	//http.Handler
	defaultCtx string
	restMap    map[string]http.Handler // key= context 名称
	restConfigMap map[string]*configs.RestConfig
}

func (mh MyHttpHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	fmt.Println("请求路径", req.URL.Path, req.RequestURI)
	fmt.Println("打印头", req.Header)
	// 处理自定义资源，如果返回true 代表被我们自己拦截，不需要再请求真实 k8s
	if NewMyResource(req, writer).HandlerForCluster(mh.restConfigMap) {
		fmt.Println("true")
		return
	}
	//解析集群参数
	cluster := helper.ParseCluster(req) // cluster 可能是空
	fmt.Println("[", cluster)
	if cluster == "" {
		cluster = mh.defaultCtx
	}
	fmt.Println(mh.defaultCtx)
	req.Header.Add("from_cluster", cluster)
	mh.restMap[cluster].ServeHTTP(writer, req) // 已经发送 kubectl
	//mh.Handler.ServeHTTP(writer, req)

}

//func WrapperHandler(h http.Handler) *MyHttpHandler {
//	return &MyHttpHandler{h}
//}

type MyTransport struct {
	http.RoundTripper
}

func (mrt MyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Println("新版拦截", req.RequestURI)         // 响应内容
	rsp, err := mrt.RoundTripper.RoundTrip(req) // 原有得执行 客户端向服务端发送请求
	if err != nil {
		return nil, err
	}
	// 获取到响应内容
	defer rsp.Body.Close()
	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		log.Fatalf("%s", err.Error())
		return nil, err
	}
	obj := &unstructured.Unstructured{}
	if err = obj.UnmarshalJSON(b); err == nil {
		// 给 结果加列  只针对 Table 加 Cluster 列
		helper.AddCustomColumn(obj, req)
		// 显示自定义内容
		helper.HandlerMyResource(obj,req)
		b, err = obj.MarshalJSON()
		if err != nil {
			return nil, err
		}
	}

	rsp.Body = ioutil.NopCloser(bytes.NewReader(b)) // 防止 关闭后没有数据
	rsp.ContentLength = int64(len(b))
	// 这句很重要，否则 kubectl会报错
	rsp.Header.Set("Content-Length", strconv.Itoa(len(b)))
	return rsp, nil
}
func WrapperTransport(h http.RoundTripper) *MyTransport {
	return &MyTransport{h}
}

func NewProxyHandler(apiProxyPrefix string, filter *FilterServer, cfg *rest.Config, keepalive time.Duration) (http.Handler, error) {
	host := cfg.Host // 获取 k8s 配置文件 host 地址
	//fmt.Println("host", host)
	if !strings.HasSuffix(host, "/") { // 判断 host 是否以 / 结尾
		host = host + "/"
	}
	target, err := url.Parse(host) // 将string解析成*URL格式
	fmt.Println(target)
	if err != nil {
		return nil, err
	}

	responder := &responder{}
	my_transport, err := rest.TransportFor(cfg)
	//fmt.Println("my_transport", my_transport)
	if err != nil {
		return nil, err
	}
	upgradeTransport, err := makeUpgradeTransport(cfg, keepalive)
	if err != nil {
		return nil, err
	}
	//创建 *UpgradeAwareHandler 对象
	my_proxy := NewUpgradeAwareHandler(target, WrapperTransport(my_transport), false, false, responder)
	my_proxy.UpgradeTransport = upgradeTransport
	my_proxy.UseRequestLocation = true

	proxyServer := http.Handler(my_proxy)
	if filter != nil {
		proxyServer = filter.HandlerFor(proxyServer)
	}

	if !strings.HasPrefix(apiProxyPrefix, "/api") { // apiProxyPrefix 是不是以 /api 开头
		proxyServer = stripLeaveSlash(apiProxyPrefix, proxyServer)
	}

	return proxyServer, nil
}

func NewServerForMultiCluster(filebase string, apiProxyPrefix string, staticPrefix string, filter *FilterServer, keepalive time.Duration) (*Server, error) {

	mux := http.NewServeMux()
	CluserterService := configs.NewClusterService() // 初始化 集群配置对象

	// 获取到 context 和 rest.config 对应关系
	restConfigMap := CluserterService.GetContextRestConfigMap()
	restMap := map[string]http.Handler{}
	defaultCtx := ""
	for ctxname, restC := range restConfigMap {
		fmt.Println("===", ctxname)
		//proxyPrefix := fmt.Sprintf("/%s/", ctxname)
		//proxyHandler, err := NewProxyHandler(apiProxyPrefix, filter, restC.RestCfg, keepalive)
		proxyHandler, err := NewProxyHandler(apiProxyPrefix, filter, restC.RestCfg, keepalive)
		if err != nil {
			continue
		}
		if restC.IsDefault { // 是否是默认集群
			defaultCtx = ctxname
		}

		restMap[ctxname] = proxyHandler
		//mux.Handle(apiProxyPrefix, WrapperHandler(proxyHandler))

	}
	mux.Handle(apiProxyPrefix, &MyHttpHandler{restMap: restMap, defaultCtx: defaultCtx,restConfigMap: restConfigMap})
	if filebase != "" {
		// Require user to explicitly request this behavior rather than
		// serving their working directory by default.
		mux.Handle(staticPrefix, newFileHandler(staticPrefix, filebase))
	}
	return &Server{handler: mux}, nil
}

//把自定义资源 做了一些封装。 因为后面要支持多个
type MyResource struct {
	req    *http.Request
	writer http.ResponseWriter
}

func NewMyResource(req *http.Request, writer http.ResponseWriter) *MyResource {
	return &MyResource{req: req, writer: writer}
}


// 这个函数是给 MyResource.HandlerForCluster (H是大写的哟)调用的
func (my *MyResource) handlerForCluster(clusters []string) []byte {
	r := regexp.MustCompile(configs.MyCluster_pattern_handler)
	if r.MatchString(my.req.RequestURI) && my.req.Method == "GET" {
		// 构建一个 unstructured
		ret := &unstructured.UnstructuredList{
			Items: make([]unstructured.Unstructured, len(clusters)),
		}
		ret.SetKind(configs.MyClusterListKind)
		ret.SetAPIVersion(configs.MyResourceApiVersion)

		for i, cluster := range clusters {
			obj := unstructured.Unstructured{}
			obj.SetAPIVersion(configs.MyResourceApiVersion)
			obj.SetKind(configs.MyClusterKind)
			obj.SetName(cluster)
			obj.SetCreationTimestamp(metav1.NewTime(time.Now()))
			ret.Items[i] = obj
		}

		b, err := ret.MarshalJSON()
		if err != nil {
			log.Println(err)
			return nil
		}
		return b
	}
	return nil
}

// 响应资源  --- 在server.go 拦截 ,目前只处理 集群列表 ，后面肯定还要加进去
// 返回值 bool，代表是否拦截到 。 如果是TRUE 外部则不应该继续响应
func (my *MyResource) HandlerForCluster(clusterMap map[string]*configs.RestConfig) bool {
	clusterNames := []string{}
	for k, _ := range clusterMap {
		clusterNames = append(clusterNames, k)
	}
	myres := my.handlerForCluster(clusterNames)
	if myres != nil {
		my.writer.Header().Set("Content-type", "application/json")
		my.writer.Header().Set("Content-Length", strconv.Itoa(len(myres)))
		my.writer.Write(myres)
		return true
	}
	return false
}
