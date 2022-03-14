package errors

import "fmt"

var (
	ErrPermissionFailed = fmt.Errorf("失败 无权进行此操作")
)
