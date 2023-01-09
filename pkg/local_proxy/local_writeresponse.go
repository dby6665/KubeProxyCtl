package local_proxy

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"strconv"
)

func WriteResponse(writer http.ResponseWriter, src []*MyWriter) {
	writer.Header().Set("Content-type", "application/json")

	if len(src) == 1 { //只有一个 说明不是 联合集群
		writer.Header().Set("Content-Length",
			strconv.Itoa(len(src[0].Content)))
		writer.Write(src[0].Content)
	} else if len(src) > 1 {
		allContent := [][]byte{}
		for _, w := range src {
			allContent = append(allContent, w.Content)
		}
		b, err := mergeResponse(allContent...)
		if err != nil {
			return
		}
		writer.Header().Set("Content-Length", strconv.Itoa(len(b)))
		writer.Write(b)
	}
}

// 目前 只处理 Table 内容
// len(cnts) 一定>0 否则不会 调用此函数
func mergeResponse(cnts ...[]byte) ([]byte, error) {
	var ret *metav1.Table

	for _, cnt := range cnts {
		tmp := &unstructured.Unstructured{}
		if err := tmp.UnmarshalJSON(cnt); err == nil {
			if tmp.GetKind() == "Table" { // 后面要改。 因为这个类型 不支持 client-go
				tb := &metav1.Table{}
				err = runtime.DefaultUnstructuredConverter.
					FromUnstructured(tmp.Object, tb)
				if err != nil {
					continue
				}
				if ret == nil {
					ret = tb
				} else {
					ret.Rows = append(ret.Rows, tb.Rows...)
				}
			}
		} else {
			//	log.Println("错误是", string(cnt), "aaaa", len(cnt))
		}
	}
	if ret == nil {
		return cnts[0], nil //  临时处理。 代表没有解析到， 临时返回第一个
	} else {
		return json.Marshal(ret)

	}

}
