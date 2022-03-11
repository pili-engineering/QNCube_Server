package maxim

import (
	"fmt"
	"github.com/tidwall/gjson"
)

type CallError struct {
	Api string
	Err error
}

func NewCallError(api string, err error) *CallError {
	return &CallError{Api: api, Err: err}
}

func (c CallError) Error() string {
	return fmt.Sprint("call api %v error:%v", c.Api, c.Err)
}

type StatusCodeError struct {
	Code int
	Msg  string
}

func NewStatusCodeError(code int, msg string) *StatusCodeError {
	return &StatusCodeError{Code: code, Msg: msg}
}

func (s StatusCodeError) Error() string {
	return fmt.Sprint("resp status %v", s.Code)
}

// im-demo used wrapper error
var (
	ErrNoSuchRtcApp                       = fmt.Errorf("not such rtc app")
	ErrUserUnqualified                    = fmt.Errorf("external service report unquaified")
	ErrUserWithoutEnterpriseCertification = fmt.Errorf("user havn't pass enter enterprise certification")
	ErrNoCorrespondIMApp                  = fmt.Errorf("no correspont im app")

	ErrBadToken = fmt.Errorf("bad token")
)

type MaximError struct {
	Code    int
	Message string
}

// NewMaximError parse and search maxim error
func NewMaximError(val []byte) *MaximError {
	result := gjson.ParseBytes(val)
	code := int(result.Get("code").Int())
	message := result.Get("message").String()
	return &MaximError{
		Code:    code,
		Message: message,
	}
}

func (m MaximError) Error() string {
	switch m.Code {
	case 10004:
		return "user has already existed"
	default:
		return fmt.Sprintf("unknow error code: %d message: %s", m.Code, m.Message)
	}
}

// error from meixin-service

var (
	ErrMeixinUserAlreadyExists = fmt.Errorf("user has already existed")
)

const (
	ErrMeixinUserAlreadyExistsCode = 10004
)
