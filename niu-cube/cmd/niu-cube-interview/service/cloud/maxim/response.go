package maxim

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func NewResponse(code int, msg string) *Response {
	return &Response{Code: code, Msg: msg}
}

func (r *Response) With(data interface{}) *Response {
	r.Data = data
	return r
}

func NewFailResponse(err error) *Response {
	return NewResponse(1, "failure").With(err.Error())
}

func NewSuccessResponse(data interface{}) *Response {
	return NewResponse(0, "success").With(data)
}
