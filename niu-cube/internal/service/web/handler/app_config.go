package handler

import (
	"net/http"

	"github.com/qiniu/x/xlog"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/form"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/cloud"

	"github.com/gin-gonic/gin"
)

type AppConfigApiHandler struct {
}

func (h *AppConfigApiHandler) GetAppConfig(c *gin.Context) {

	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: model.AppConfigResponse{
			WelcomeResponse: model.WelcomeResponse{
				Image: utils.DefaultConf.WelcomeImage,
				Url:   utils.DefaultConf.WelcomeURL,
			},
		},
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AppConfigApiHandler) SolutionList(c *gin.Context) {
	mobileOs, isMobile := c.Get(model.UAContextKey)
	apiVersion, hasApiVersion := c.Get(model.RequestApiVersion)
	solutionResponseList := utils.DefaultConf.Solutions
	if isMobile && mobileOs == model.UAMobileApple && hasApiVersion && apiVersion == model.ApiVersionV1 {
		solutionResponseList = utils.DefaultConf.Solutions4Apple
	} else if isMobile && mobileOs == model.UAMobileAndroid && hasApiVersion && apiVersion == model.ApiVersionV1 {
		solutionResponseList = utils.DefaultConf.Solutions4Android
	}
	var solutionList = make([]interface{}, len(solutionResponseList))
	for index, solutionObj := range solutionResponseList {
		solutionList[index] = solutionObj
	}

	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: model.SolutionListResponse{
			Pagination: model.Pagination{
				CurrentPageNum: 1,
				NextPageNum:    1,
				PageSize:       10,
				EndPage:        true,
				Total:          4,
				NextId:         "",
				Cnt:            4,
				List:           solutionList,
			},
		},
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AppConfigApiHandler) KodoToken(c *gin.Context) {
	token := cloud.GenkodoClientToken(utils.DefaultConf.QiniuKeyPair, utils.DefaultConf.Weixin.Bucket)
	res := map[string]interface{}{
		"token": token,
	}
	c.JSON(http.StatusOK, res)
	return
}

func (h *AppConfigApiHandler) GetToken(c *gin.Context) {

	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	args := form.TokenForm{}
	c.Bind(&args)
	content := args.Content
	xl.Warnf("content:%v", content)
	token := cloud.GetToken(utils.DefaultConf.QiniuKeyPair, content)
	data := map[string]interface{}{
		"token": token,
	}

	// 返回值
	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      data,
		RequestID: requestID,
	}
	c.JSON(http.StatusOK, resp)
	return
}
