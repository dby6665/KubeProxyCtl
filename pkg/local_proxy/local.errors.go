package local_proxy

/**
* @Author: DK
* @Date: 2022/12/4 16:57
* Description: 描述
* Updated:时间@版本@变更说明
 */


// New returns an error that formats as the given text.
// Each call to New returns a distinct error value even if the text is identical.
func New(text string) error {
	return &errorString{text}
}

// errorString is a trivial implementation of error.
type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}