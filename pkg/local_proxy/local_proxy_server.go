package local_proxy

import (
	"fmt"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

/**
* @Author: DK
* @Date: 2022/12/2 10:59
* Description: 原 proxy_server 文件内容
* Updated:时间@版本@变更说明
 */

// 过滤服务
type FilterServer struct {
	// Only paths that match this regexp will be accepted
	AcceptPaths []*regexp.Regexp
	// Paths that match this regexp will be rejected, even if they match the above
	RejectPaths []*regexp.Regexp
	// Hosts are required to match this list of regexp
	AcceptHosts []*regexp.Regexp
	// Methods that match this regexp are rejected
	RejectMethods []*regexp.Regexp
	// The delegate to call to handle accepted requests.
	delegate http.Handler
}

type Server struct {
	handler http.Handler
}
type responder struct{}

func (r *responder) Error(w http.ResponseWriter, req *http.Request, err error) {
	klog.Errorf("Error while proxying request: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
func (f *FilterServer) accept(method, path, host string) bool {
	fmt.Println("method",method,f.RejectMethods)
	fmt.Println("path",path,f.AcceptPaths)
	fmt.Println("host",host,f.AcceptHosts)
	if matchesRegexp(path, f.RejectPaths) {
		fmt.Println("path")
		return false
	}
	if matchesRegexp(method, f.RejectMethods) {
		fmt.Println("method")
		return false
	}
	if matchesRegexp(path, f.AcceptPaths) && matchesRegexp(host, f.AcceptHosts) {
		fmt.Println("path,host")
		return true
	}
	return false
}

// HandlerFor makes a shallow copy of f which passes its requests along to the
// new delegate.
func (f *FilterServer) HandlerFor(delegate http.Handler) *FilterServer {
	f2 := *f
	f2.delegate = delegate
	return &f2
}

// Get host from a host header value like "localhost" or "localhost:8080"
func extractHost(header string) (host string) {
	host, _, err := net.SplitHostPort(header)
	if err != nil {
		host = header
	}
	return host
}

func (f *FilterServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	host := extractHost(req.Host)
	if f.accept(req.Method, req.URL.Path, host) {
		klog.V(3).Infof("Filter accepting %v %v %v", req.Method, req.URL.Path, host)
		f.delegate.ServeHTTP(rw, req)
		return
	}
	klog.V(3).Infof("Filter rejecting %v %v %v", req.Method, req.URL.Path, host)
	http.Error(rw, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}
func matchesRegexp(str string, regexps []*regexp.Regexp) bool {
	fmt.Println("str",str,regexps)
	for _, re := range regexps {
		if re.MatchString(str) {
			klog.V(6).Infof("%v matched %s", str, re)
			return true
		}
	}
	return false
}
func makeUpgradeTransport(config *rest.Config, keepalive time.Duration) (proxy.UpgradeRequestRoundTripper, error) {
	transportConfig, err := config.TransportConfig()
	if err != nil {
		return nil, err
	}
	tlsConfig, err := transport.TLSConfigFor(transportConfig)
	if err != nil {
		return nil, err
	}
	rt := utilnet.SetOldTransportDefaults(&http.Transport{
		TLSClientConfig: tlsConfig,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: keepalive,
		}).DialContext,
	})

	upgrader, err := transport.HTTPWrappersForConfig(transportConfig, proxy.MirrorRequest)
	if err != nil {
		return nil, err
	}
	return proxy.NewUpgradeRequestRoundTripper(rt, upgrader), nil
}
func stripLeaveSlash(prefix string, h http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fmt.Println(req.URL.Path, prefix)
		p := strings.TrimPrefix(req.URL.Path, prefix)
		fmt.Println(len(p))
		if len(p) >= len(req.URL.Path) {

			http.NotFound(w, req)
			return
		}
		if len(p) > 0 && p[:1] != "/" {
			p = "/" + p
		}
		fmt.Println(p)
		req.URL.Path = p
		h.ServeHTTP(w, req)
	})
}

func newFileHandler(prefix, base string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.Dir(base)))
}
func MakeRegexpArrayOrDie(str string) []*regexp.Regexp {
	result, err := MakeRegexpArray(str)
	if err != nil {
		klog.Fatalf("Error compiling re: %v", err)
	}
	return result
}
func MakeRegexpArray(str string) ([]*regexp.Regexp, error) {
	parts := strings.Split(str, ",")
	result := make([]*regexp.Regexp, len(parts))
	for ix := range parts {
		re, err := regexp.Compile(parts[ix])
		if err != nil {
			return nil, err
		}
		result[ix] = re
	}
	fmt.Println("result",result)
	return result, nil
}




func NewServer(filebase string, apiProxyPrefix string, staticPrefix string, filter *FilterServer, cfg *rest.Config, keepalive time.Duration) (*Server, error) {
	host := cfg.Host  // 获取 k8s 配置文件 host 地址
	if !strings.HasSuffix(host, "/") {
		host = host + "/"
	}
	target, err := url.Parse(host) // 将string解析成*URL格式
	if err != nil {
		return nil, err
	}

	responder := &responder{}
	//local_transport, err := rest.TransportFor(cfg)
	local_transport, err := TransportFor(cfg)
	//fmt.Println("local_transport",local_transport)
	if err != nil {
		return nil, err
	}
	upgradeTransport, err := makeUpgradeTransport(cfg, keepalive) //
	if err != nil {
		return nil, err
	}
	//创建 *UpgradeAwareHandler 对象
	local_proxy := NewUpgradeAwareHandler(target, WrapperTransport(local_transport), false, false, responder)
	local_proxy.UpgradeTransport = upgradeTransport
	local_proxy.UseRequestLocation = true


	proxyServer := http.Handler(local_proxy) // http.Handler
	if filter != nil {
		proxyServer = filter.HandlerFor(proxyServer)
	}

	if !strings.HasPrefix(apiProxyPrefix, "/api") {

		proxyServer = stripLeaveSlash(apiProxyPrefix, proxyServer)
	}

	mux := http.NewServeMux() // 创建空的 ServerMux
	//fmt.Println(apiProxyPrefix,proxyServer)
	//mux.Handle(apiProxyPrefix, WrapperHandler(proxyServer)) // 路由注册到 ServerMux ，路径 apiProxyPrefix 上收到的所有请求都会交给proxyServer处理器
	mux.Handle(apiProxyPrefix, proxyServer)// 路由注册到 ServerMux ，路径 apiProxyPrefix 上收到的所有请求都会交给proxyServer处理器
	if filebase != "" {
		// Require user to explicitly request this behavior rather than
		// serving their working directory by default.
		mux.Handle(staticPrefix, newFileHandler(staticPrefix, filebase))
	}
	return &Server{handler: mux}, nil
}
// Listen is a simple wrapper around net.Listen.
func (s *Server) Listen(address string, port int) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
}

// ListenUnix does net.Listen for a unix socket
func (s *Server) ListenUnix(path string) (net.Listener, error) {
	// Remove any socket, stale or not, but fall through for other files
	fi, err := os.Stat(path)
	if err == nil && (fi.Mode()&os.ModeSocket) != 0 {
		os.Remove(path)
	}
	// Default to only user accessible socket, caller can open up later if desired
	oldmask, _ := util.Umask(0077)
	l, err := net.Listen("unix", path)
	util.Umask(oldmask)
	return l, err
}

// ServeOnListener starts the server using given listener, loops forever.
func (s *Server) ServeOnListener(l net.Listener) error {
	server := http.Server{
		Handler: s.handler,
	}
	return server.Serve(l)
}