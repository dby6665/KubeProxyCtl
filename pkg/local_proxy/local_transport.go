package local_proxy

import (
	"fmt"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
	"net/http"
)

/**
* @Author: DK
* @Date: 2022/12/2 11:17
* Description: 描述
* Updated:时间@版本@变更说明
 */
type Transport struct {
	Scheme      string
	Host        string
	PathPrepend string

	http.RoundTripper
}


var _ fmt.Stringer = new(rest.Config)
var _ fmt.GoStringer = new(rest.Config)


func TransportFor(config *rest.Config) (http.RoundTripper, error) {
	//fmt.Println("transport configs",config)
	cfg, err := config.TransportConfig()
	if err != nil {
		return nil, err
	}
	return transport.New(cfg)
}
