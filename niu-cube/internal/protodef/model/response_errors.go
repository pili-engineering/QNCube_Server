package model

type ResponseError struct {
	// 自定义错误码。
	Code int `json:"code"`
	// 请求ID。
	RequestID string `json:"requestID"`
	// Message
	Message string `json:"message"`
}

const (
	ResponseErrorBadRequest         = 400000
	ResponseErrorNotLoggedIn        = 401001
	ResponseErrorWrongSMSCode       = 401002
	ResponseErrorBadToken           = 401003
	ResponseErrorAlreadyLoggedIn    = 401004
	ResponseErrorNoSuchUser         = 404001
	ResponseErrorNoSuchInterview    = 404002
	ResponseErrorNoSuchBoard        = 404003
	ResponseErrorNoSuchRoom         = 404004 // TODO: add to doc
	ResponseErrorSMSSendTooFrequent = 429001
	ResponseErrorInternal           = 500000
	ResponseErrorExternalService    = 502001
	ResponseErrorUnauthorized       = 401000
	ResponseErrorNotFound           = 404000
	ResponseErrorValidation         = 401005
	ResponseErrorJoinRoom           = 401006
	ResponseErrorValidationRoomId   = 401007
	ResponseErrorGetRoomContent     = 401008
	ResponseErrorOnlyOneStaff       = 401009
	ResponseErrorTooManyPeople      = 401010
	ResponseErrorExamTimeNotMatch   = 401011
	ResponseErrorExamDuplicateEntry = 401012
)

// NewHTTPErrorBadRequest 参数错误。
func NewResponseErrorBadRequest() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorBadRequest,
		Message: "参数错误",
	}
}

// NewResponseErrorNotLoggedIn 用户未登录。
func NewResponseErrorNotLoggedIn() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorNotLoggedIn,
		Message: "not logged in",
	}
}

// NewResponseErrorWrongSMSCode 用户短信验证码错误。
func NewResponseErrorWrongSMSCode() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorWrongSMSCode,
		Message: "wrong sms code",
	}
}

// NewResponseErrorBadToken 登录token错误。
func NewResponseErrorBadToken() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorBadToken,
		Message: "bad token",
	}
}

// NewResponseErrorSMSSendTooFrequent 短信验证码已发送，短时间内不能重复发送。
func NewResponseErrorSMSSendTooFrequent() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorSMSSendTooFrequent,
		Message: "send sms code request limited",
	}
}

// NewResponseErrorInternal 其他内部服务错误。
func NewResponseErrorInternal() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorInternal,
		Message: "internal server error",
	}
}

// NewResponseErrorAlreadyLoggedin 用户已经登录，此为重复登录
func NewResponseErrorAlreadyLoggedin() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorAlreadyLoggedIn,
		Message: "already logged in",
	}
}

// NewResponseErrorExternalService 调用外部服务错误。
func NewResponseErrorExternalService() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorExternalService,
		Message: "calling external service failed",
	}
}

// NewResponseErrorNoSuchUser 无此用户。
func NewResponseErrorNoSuchUser() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorNoSuchUser,
		Message: "no such user",
	}
}

// NewResponseErrorUnauthorized 一般的HTTP Unauthorized 错误。
func NewResponseErrorUnauthorized() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorUnauthorized,
		Message: "unauthorized",
	}
}

func NewResponseErrorNotFound() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorNotFound,
		Message: "not found",
	}
}

// NewResponseErrorNoSuchInterview 无此房间。
func NewResponseErrorNoSuchInterview() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorNoSuchInterview,
		Message: "no such interview",
	}
}

func NewResponseErrorValidation(err error) *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorValidation,
		Message: err.Error(),
	}
}

func NewResponseErrorNoSuchBoard() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorNoSuchBoard,
		Message: "no such board",
	}
}

func NewResponseErrorNoSuchRoom() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorNoSuchRoom,
		Message: "not such room",
	}
}

func NewResponseErrorJoinRoom() *ResponseError {
	return &ResponseError{
		Code:    ResponseErrorJoinRoom,
		Message: "join room fail",
	}
}

func NewResponseError(code int, message string) *ResponseError {
	return &ResponseError{
		Code:    code,
		Message: message,
	}
}
