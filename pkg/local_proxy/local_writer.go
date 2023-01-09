package local_proxy

import "net/http"

func WrapWriter() *MyWriter {

	return &MyWriter{Content: make([]byte, 0)}
}

type MyWriter struct {
	http.ResponseWriter
	HeaderCode int
	//content    [][]byte //保存内容
	Content []byte //保存内容
}

func (w *MyWriter) Header() http.Header {
	return map[string][]string{
		"Content-type": []string{"application/json"},
	}

}
func (w *MyWriter) Write(b []byte) (int, error) {

	//w.content = append(w.content, b)
	w.Content = append(w.Content, b...) /// 合并一个   因为每一个 请求 都是独立的writer
	return len(b), nil
}

func (w *MyWriter) WriteHeader(h int) {
	w.HeaderCode = h
	return
}
