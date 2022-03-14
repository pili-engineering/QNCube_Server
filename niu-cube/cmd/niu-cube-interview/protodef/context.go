package protodef

const (
	ContextUserKey   = "user"
	ContextUserIdKey = "userId"

	ContextCandidateKey   = "candidate"
	ContextCandidateIdKey = "candidateId"
	// for mock use

	ParamPathInterviewId = "interviewId"
	// path param

	// RequestIDHeader 七牛 request ID 头部。
	RequestIDHeader = "X-Reqid"
	// XLogKey gin context中，用于获取记录请求相关日志的 xlog logger的key。
	XLogKey = "xlog-logger"

	HeaderTokenKey = "Authorization"
)
