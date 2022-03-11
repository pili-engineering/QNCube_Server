package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/dao"
	"net/http"
)

type AppVersionApi interface {
	UpdateAppVersion(context *gin.Context)

	GetNewestAppVersion(context *gin.Context)
}

type AppVersionApiHandler struct {
	appVersionDao dao.AppVersionDao
}

func NewAppVersionApiHandler(config *utils.MongoConfig) *AppVersionApiHandler {
	return &AppVersionApiHandler{
		appVersionDao: dao.NewAppVersionDaoService(config),
	}
}

func (a *AppVersionApiHandler) UpdateAppVersion(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	var req model.AppVersion
	if err := context.Bind(&req); err != nil {
		xl.Infof("invalid args in body, error: %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	err := a.appVersionDao.InsertAppVersion(&req)
	if err != nil {
		xl.Errorf("insert app version error: %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestId)
		context.JSON(http.StatusOK, resp)
		return
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
			Result bool `json:"result"`
		}{
			Result: true,
		},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
	return
}

func (a *AppVersionApiHandler) GetNewestAppVersion(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	version := context.Query("version")
	arch := context.Query("arch")
	appVersion, err := a.appVersionDao.GetNewestAppVersion(arch)
	if err != nil {
		xl.Errorf("get newest app version error: %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(xl.ReqId)
		context.JSON(http.StatusOK, resp)
		return
	}
	if appVersion.Version == version {
		resp := &model.Response{
			Code:    int(model.ResponseStatusCodeSuccess),
			Message: string(model.ResponseStatusMessageSuccess),
			Data: model.AppVersion{
				Version:     appVersion.Version,
				Msg:         appVersion.Msg,
				PackagePage: "",
				PackageUrl:  "",
			},
			RequestID: xl.ReqId,
		}
		context.JSON(http.StatusOK, resp)
		return
	} else {
		resp := &model.Response{
			Code:      int(model.ResponseStatusCodeSuccess),
			Message:   string(model.ResponseStatusMessageSuccess),
			Data:      appVersion,
			RequestID: xl.ReqId,
		}
		context.JSON(http.StatusOK, resp)
		return
	}
}
