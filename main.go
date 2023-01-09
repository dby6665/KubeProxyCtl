package main

import (
	"crypto/tls"
	"fmt"
	"kubeProxyCtl/pkg/local_proxy"
	"kubeProxyCtl/tools/utils/configs"
	"log"
)

/**
* @Author: DK
* Description: 启动程序

 */
// local_proxy.MakeRegexpArrayOrDie("^/api/.*/pods,^/api/.*/pods/.*/attach")
const (
	// DefaultHostAcceptRE is the default value for which hosts to accept.
	DefaultHostAcceptRE = "^localhost$,^127\\.0\\.0\\.1$,^\\[::1\\]$"
	// DefaultPathAcceptRE is the default path to accept.
	DefaultPathAcceptRE = "^/qpi"
	// DefaultPathRejectRE is the default set of paths to reject.
	DefaultPathRejectRE = "^/api/.*/pods/.*/exec,^/api/.*/pods/.*/attach"
	// DefaultMethodRejectRE is the set of HTTP methods to reject by default.
	DefaultMethodRejectRE = "^$"
)

func main() {
	// k get pods --selector "cluster==tx"
	// kubectl --kubeconfig config get jc
	//filter := &local_proxy.FilterServer{
	//	AcceptPaths:   local_proxy.MakeRegexpArrayOrDie(DefaultPathAcceptRE),
	//	RejectPaths:   local_proxy.MakeRegexpArrayOrDie(DefaultPathRejectRE),
	//	AcceptHosts:   local_proxy.MakeRegexpArrayOrDie(DefaultHostAcceptRE),
	//	RejectMethods: local_proxy.MakeRegexpArrayOrDie(DefaultMethodRejectRE),
	//}

	//restConfig := configs.NewK8sConfig().K8sRestConfigDefault()

	//server, err := local_proxy.NewServer("", configs.DefaultApiProxyPrefix,
	//	"", nil, restConfig, 0)
	server, err := local_proxy.NewServerForMultiCluster(configs.Default, configs.DefaultApiProxyPrefix, configs.Default, nil, 0)
	if err != nil {
		log.Fatalln(err)
	}
	// 配置证书
	cert, err := tls.LoadX509KeyPair(fmt.Sprintf("%s", configs.CertFile),
		fmt.Sprintf("%s", configs.KeyFile))
	if err != nil {
		log.Fatal(err)
	}
	tlsConfig := tls.Config{Certificates: []tls.Certificate{cert}}
	addr := fmt.Sprintf("%s:%s", configs.ProjectDomain, configs.ProjectPort)
	listener, err := tls.Listen("tcp", addr, &tlsConfig)
	//l, err := sever.Listen("0.0.0.0", 8919)
	if err != nil {
		log.Fatal(listener)
		return
	}
	//err = sever.ServeOnListener(l)
	//if err != nil {
	//	log.Fatalln(err)
	//}
	if err = server.ServeOnListener(listener); err != nil {
		log.Fatal(err)
	}

	//l, err := server.Listen("0.0.0.0", 8919)
	//err = server.ServeOnListener(l)
	//if err != nil {
	//	log.Fatalln(err)
	//}
}
