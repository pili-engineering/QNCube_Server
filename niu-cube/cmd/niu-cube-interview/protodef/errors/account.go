package errors

const (
	ErrEmailFormatMsg          = "邮箱格式不规范"
	ErrEmailOrPhoneRequiredMsg = "邮箱、手机号必填其一"
)

var (
	ErrEmailOrPhoneRequired = NewFormValidationError(ErrEmailOrPhoneRequiredMsg)
)
