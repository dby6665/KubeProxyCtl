package helper

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"kubeProxyCtl/tools/utils/configs"
	"net/http"
	"regexp"
	"strings"
)

func ParseCluster(req *http.Request) string {

	cluster := ""
	newSelector := []string{}

	// pods?labelSelector=cluster%3D%3Dhw&limit=500
	values := req.URL.Query() //保存个副本
	//解析完成后，要去掉cluster=xxx  ， 重新设置 参数，否则会查不到
	if selector := req.URL.Query().Get(configs.ClusterParam); selector != "" {
		//按逗号切割
		strSplit := strings.Split(selector, ",") //  --selector "cluster==hw,app=nginx"
		for _, param := range strSplit {
			if c := parseSelectorIfCluster(param); c != "" {
				cluster = c
				continue
			}
			newSelector = append(newSelector, param) //把不是cluster的参数 依然加入
		}
		//更新request的url --- 排除cluster标记

		values.Set(configs.ClusterParam, strings.Join(newSelector, ","))

		//fmt.Println(req.URL.RawQuery)
		req.URL.RawQuery = values.Encode()
	}
	return cluster
}

// 解析 app=ngx,cluster=xxx 的字符串
// string 是cluster的值
func parseSelectorIfCluster(param string) string {
	pair := strings.Split(param, "==") // 此处在win上就必须是== 在linux上== 和= 都可以 ，后面可以改成正则
	if len(pair) == 2 {
		if pair[0] == configs.ClusterKey {
			return pair[1]
		}
	}
	return ""
}

// 针对table类型  加入集群标志
func AddCustomColumn(obj *unstructured.Unstructured, req *http.Request) {
	//tb := metav1.Table{}
	if obj.GetKind() == "Table" {
		if cd, ok := obj.Object["columnDefinitions"].([]interface{}); ok {
			cd = append(cd, map[string]interface{}{
				"name":        "cluster",
				"description": "集群",
				"format":      "name",
				"type":        "string",
				"priority":    0,
			})
			obj.Object["columnDefinitions"] = cd
		}

		if rows, ok := obj.Object["rows"].([]interface{}); ok {
			newRows := []interface{}{}
			for _, row := range rows {
				r := row.(map[string]interface{})
				if cells, ok := r["cells"].([]interface{}); ok {
					cells = append(cells, req.Header.Get("from_cluster"))
					row.(map[string]interface{})["cells"] = cells
				}
				newRows = append(newRows, row)

			}
			// obj 非结构化的 本质就是 map[string]interface
			obj.Object["rows"] = newRows

		}
	}
}

// 放内部资源
// 目前 放到  /apps/v1里
//拦截自定义资源 做修改  ---在transport.go 里拦截
func HandlerMyResource(obj *unstructured.Unstructured, req *http.Request) {
	r := regexp.MustCompile(configs.Myres_pattern)
	if r.MatchString(req.RequestURI) && obj.GetKind() == "APIResourceList" {
		if resList, ok := obj.Object["resources"].([]interface{}); ok {
			if !existsClusterDef(resList) {
				resList = append(resList, getClusterMap())
				obj.Object["resources"] = resList
			}

		}
	}
}

//是否已经定义过 自定义资源
func existsClusterDef(resList []interface{}) bool {
	for _, res := range resList {
		if r, ok := res.(map[string]interface{}); ok {
			if n, ok := r["name"]; ok && n == configs.MyClusterName {
				return true
			}
		}
	}
	return false
}


func getClusterMap() map[string]interface{} {
	return map[string]interface{}{
		"kind":               configs.MyClusterKind,
		"name":               configs.MyClusterName,
		"namespaced":         true,
		"singularName":       "",
		"storageVersionHash": "",
		"categories":         []string{"all"},
		"shortNames":         []string{configs.MyClusterShortName},
		"verbs":              []string{"get", "list", "patch"},
	}
}
