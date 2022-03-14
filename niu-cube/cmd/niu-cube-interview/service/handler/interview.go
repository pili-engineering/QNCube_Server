package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef/form"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef/model"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/service/cloud"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/service/db"
	"net/http"
)

type InterviewHandler interface {
	Create(c *gin.Context)
	Join(c *gin.Context)
	Leave(c *gin.Context)
	Cancel(c *gin.Context)
	End(c *gin.Context)
	Share(c *gin.Context)
	Update(c *gin.Context)
	List(c *gin.Context)

	RegisterRoute(group *gin.RouterGroup)
}

type InterviewHandlerImpl struct {
	interviewService db.InterviewService
	imService        cloud.IMService
	xl               *xlog.Logger
}

func NewInterviewHandle(interviewService db.InterviewService) InterviewHandler {
	return &InterviewHandlerImpl{interviewService: interviewService, xl: xlog.New("interview handler")}
}

func (i InterviewHandlerImpl) RegisterRoute(group *gin.RouterGroup) {
	group.POST("interview/create", i.Create)
	group.POST("interview/:interviewId/join", i.Join)
	group.POST("interview/:interviewId/leave", i.Leave)
	group.POST("interview/:interviewId/cancel", i.Cancel)
	group.POST("interview/:interviewId/end", i.End)
}

func (i InterviewHandlerImpl) Create(c *gin.Context) {
	var interviewCreateForm form.InterviewCreateForm
	err := c.Bind(&interviewCreateForm)
	if err != nil {
		// error response
		i.xl.Errorf("error binding err:%v", err)
		c.JSON(http.StatusOK, protodef.BindErrResponse(err))
		return
	}
	user := c.MustGet(protodef.ContextUserKey).(model.Account)
	candidate := c.MustGet(protodef.ContextCandidateKey).(model.Account)
	err = i.interviewService.Create(user, candidate, "", interviewCreateForm.Map())
	if err != nil {
		i.xl.Errorf("error interview create service invoke err:%v", err)
		c.JSON(http.StatusOK, protodef.BindErrResponse(err))
		return
	}
	c.JSON(http.StatusOK, protodef.MockSuccessResponse("create success"))
}

func (i InterviewHandlerImpl) Join(c *gin.Context) {
	user := c.MustGet(protodef.ContextUserKey).(model.Account)
	interviewId := c.Param(protodef.ParamPathInterviewId)
	i.xl.Errorf("interviewId :%s", interviewId)
	//candidate:=c.MustGet(protodef.ContextCandidateKey).(model.Account)
	//err=i.interviewService.Create(user,candidate,"",interviewCreateForm.Map())
	err := i.interviewService.Join(user, interviewId)
	if err != nil {
		i.xl.Errorf("error interview join service invoke err:%v", err)
		c.JSON(http.StatusOK, protodef.BindErrResponse(err))
		return
	}
	c.JSON(http.StatusOK, protodef.MockSuccessResponse("join success"))
}

func (i InterviewHandlerImpl) Leave(c *gin.Context) {
	c.JSON(http.StatusOK, "implement me")
}

func (i InterviewHandlerImpl) Cancel(c *gin.Context) {
	c.JSON(http.StatusOK, "implement me")
}

func (i InterviewHandlerImpl) End(c *gin.Context) {
	c.JSON(http.StatusOK, "implement me")
}

func (i InterviewHandlerImpl) Share(c *gin.Context) {
	c.JSON(http.StatusOK, "implement me")
}

func (i InterviewHandlerImpl) Update(c *gin.Context) {
	c.JSON(http.StatusOK, "implement me")
}

func (i InterviewHandlerImpl) List(c *gin.Context) {
	c.JSON(http.StatusOK, "implement me")
}

// logger
func (i InterviewHandlerImpl) logger(c *gin.Context) *xlog.Logger {
	logger := i.xl
	val, ok := c.Get(protodef.XLogKey)
	if ok {
		logger = val.(*xlog.Logger)
	}
	return logger
}
