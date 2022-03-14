package form

import (
	"fmt"
	model "github.com/solutions/niu-cube/internal/protodef/model"
)

var (
	ErrInterviewIdNeeded = fmt.Errorf("InterviewID 是必要的")
	ErrBoardLocked       = fmt.Errorf("当前白板已锁定")
)

type BoardCreateForm struct {
	//InterviewID string `json:"interview_id" form:"interview_id" uri:"interview_id"`
	//ID string `json:"id" form:"_id"`
	//StatusCode protocol.BoardStatusCode `json:"status_code" form:"status_code"`
	Cmd model.BoardCmd `json:"cmd" form:"cmd" uri:"cmd"`
}

func (v *BoardCreateForm) Validate() error {
	err := shallowIn("cmd", v.Cmd, model.BoardCmdOpen, model.BoardCmdClose, model.BoardCmdReset)
	if err != nil {
		return err
	}
	//if v.InterviewID==""{
	//	return ErrInterviewIdNeeded
	//}
	// InterviewID 校验
	return err
}
