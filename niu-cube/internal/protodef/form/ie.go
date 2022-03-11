package form

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/solutions/niu-cube/internal/protodef/model"
)

const (
	ErrTitleLengthMsg  = "房间标题长度不应超过20个字符"
	ErrNoticeLengthMsg = "房间公告长度不应超过100个字符"
)

type IECreateForm struct {
	Title  string `json:"title" form:"title"`
	Notice string `json:"notice" form:"notice"`
}

type IEUpadteForm = IECreateForm

// Validate form validation for Ie create form
func (i *IECreateForm) Validate() error {
	if err := validation.ValidateStruct(i,
		validation.Field(&i.Title, validation.Required, validation.RuneLength(0, 20).Error(ErrTitleLengthMsg)),
		validation.Field(&i.Notice, validation.Required, validation.RuneLength(0, 100).Error(ErrNoticeLengthMsg)),
	); err != nil {
		if val, ok := err.(validation.InternalError); ok {
			panic(val)
		}
		return err
	}
	return nil
}

// FillDefault retrieve user from context, fill form default value with user
func (i *IECreateForm) FillDefault(c *gin.Context) {
	panic("need implement")
}

func (i *IECreateForm) Map() model.FlattenMap {
	val, _ := json.Marshal(i)
	res := make(map[string]interface{})
	_ = json.Unmarshal(val, &res)
	return res
}
