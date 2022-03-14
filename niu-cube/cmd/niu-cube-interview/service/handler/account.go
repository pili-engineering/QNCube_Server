package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/common"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef/form"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef/model"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/service/cloud"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/service/db"
	"net/http"
	"time"
)

type AccountHandler interface {
	Login(c *gin.Context)
	Logout(c *gin.Context)
	SignUp(c *gin.Context)

	RegisterRoute(group *gin.RouterGroup)
}

type AccountHandlerImpl struct {
	accountService db.AccountService
	imService      cloud.IMService
	xl             *xlog.Logger
}

func NewAccountHandler(accountService db.AccountService, imService cloud.IMService) AccountHandler {
	return &AccountHandlerImpl{accountService: accountService, imService: imService, xl: xlog.New("account handler")}
}

// SignUp accout/signup
func (a AccountHandlerImpl) SignUp(c *gin.Context) {
	args := form.AccountCreateForm{}
	if err := c.Bind(&args); err != nil {
		c.JSON(http.StatusOK, protodef.BindErrResponse(err))
		return
	}
	if err := args.Validate(); err != nil {
		c.JSON(http.StatusOK, protodef.ValidationErrResponse(err))
		return
	}

	// add model
	now := time.Now()
	account := model.Account{
		Phone:         args.Phone,
		Password:      args.Password,
		Nickname:      args.Nickname,
		RegisterIP:    "",
		Email:         args.Email,
		RegisterTime:  now,
		LastLoginTime: now,
		Kind:          "user",
	}
	userId, err := a.accountService.SignUp(account) //may db\permission error 统一处理
	if err != nil {
		a.logger(c).Errorf("error signup form:%v err:%v", args, err)
		c.JSON(http.StatusOK, protodef.MockFailResponse(err))
		return
	}
	data := map[string]interface{}{"token": userId}
	c.JSON(http.StatusOK, *protodef.MockSuccessResponse("注册成功").With(data))
	return
}

// Login account/login
func (a AccountHandlerImpl) Login(c *gin.Context) {
	args := form.AccountLoginForm{}
	if err := c.Bind(&args); err != nil {
		c.JSON(http.StatusOK, protodef.BindErrResponse(err))
		return
	}
	if err := args.Validate(); err != nil {
		c.JSON(http.StatusOK, protodef.ValidationErrResponse(err))
		return
	}
	user, err := a.accountService.LoginByPassword(args.Email, args.Password) //may db\permission error 统一处理
	if err != nil {
		a.logger(c).Errorf("error login form:%v err:%v", args, err)
		c.JSON(http.StatusOK, protodef.MockFailResponse(err))
		return
	}
	userD := user.Map().Filter("nickname", "phone", "email")
	authT := common.JwtSign(map[string]interface{}{"userId": user.ID})
	imT, err := a.imService.GetUserToken(a.logger(c), user.ID)
	if err != nil {
		a.logger(c).Errorf("err call im service:%v", err)
		c.JSON(http.StatusOK, protodef.ExternalServiceErrResponse(err))
		return
	}
	userD.Merge(map[string]interface{}{"loginToken": authT, "imConfig": model.MakeFlattenMap("token", imT.Token, "type", model.ImTypeRongyun)}) // TODO: define key name as const
	c.JSON(http.StatusOK, protodef.MockSuccessResponse("登录成功").With(userD))
	return
}

// Logout account/logout
func (a AccountHandlerImpl) Logout(c *gin.Context) {
	panic("implement me")
}

// logger retrieve logger from request,use handler 's logger if none in req
func (a AccountHandlerImpl) logger(c *gin.Context) *xlog.Logger {
	logger := a.xl
	val, ok := c.Get(protodef.XLogKey)
	if ok {
		logger = val.(*xlog.Logger)
	}
	return logger
}

func (a AccountHandlerImpl) RegisterRoute(group *gin.RouterGroup) {
	group.GET("account/logout", a.Logout)
	group.GET("account/login", a.Login)
	group.GET("account/signup", a.SignUp)
}
