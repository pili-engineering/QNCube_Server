package form

import (
	"fmt"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	CreateRoleMap = map[string]string{"professor": "教授", "staff": "员工"}
	JoinRoleMap   = map[string]string{"student": "学生", "professor": "教授", "staff": "员工"}
	ErrRoleMsg    = fmt.Errorf("角色值不对")
)

const (
	ErrRoomIdMsg = "房间号不合理"
)

type RepairCreateForm struct {
	Title string `form:"title"`
	Role  string `form:"role"`
}

type RepairJoinForm struct {
	RoomId string `form:"roomId"`
	Role   string `form:"role"`
}

func (i *RepairCreateForm) Validate() error {
	err := validation.ValidateStruct(i,
		validation.Field(&i.Title, validation.Required, validation.Length(0, 100).Error(ErrTitleMsg)),
		validation.Field(&i.Role, validation.Required.Error("必填")),
	)
	if err == nil {
		role := i.Role
		_, ok := CreateRoleMap[role]
		if !ok {
			return ErrRoleMsg
		}
	}
	return err
}

func (i *RepairJoinForm) Validate() error {
	err := validation.ValidateStruct(i,
		validation.Field(&i.RoomId, validation.Required, validation.Length(0, 100).Error(ErrRoomIdMsg)),
		validation.Field(&i.Role, validation.Required.Error("必填")),
	)
	if err == nil {
		role := i.Role
		_, ok := JoinRoleMap[role]
		if !ok {
			return ErrRoleMsg
		}
	}
	return err
}
