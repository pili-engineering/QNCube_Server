package protodef

var (
	UnAuthorizedResponse = NewResponse("未认证", "未认证code")
)

type Response struct {
	Msg  string
	Code interface{} // TODO: interface for mock use
	Data interface{}
}

func NewResponse(msg string, code interface{}) *Response {
	return &Response{Msg: msg, Code: code}
}

// With set Data field
func (r *Response) With(val interface{}) *Response {
	r.Data = val
	return r
}

func BindErrResponse(err error) *Response {
	return NewResponse(err.Error(), "表单绑定错误code")
}

func ValidationErrResponse(err error) *Response {
	return NewResponse(err.Error(), "表单验证错误code")

}

func MockSuccessResponse(msg string) *Response {
	return NewResponse(msg, 0)

}

func MockFailResponse(err error) *Response {
	return NewResponse(err.Error(), -1)
}

func ExternalServiceErrResponse(err error) *Response {
	return NewResponse(err.Error(), "外部服务错误依赖code")
}
