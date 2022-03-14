package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	form2 "github.com/solutions/niu-cube/internal/protodef/form"
	model "github.com/solutions/niu-cube/internal/protodef/model"
	service2 "github.com/solutions/niu-cube/internal/service/db"
	"net/http"
	"time"
)

type VersionHandlerApi struct {
	versionService *service2.VersionService
	xl             *xlog.Logger
}

func NewVersionHandlerApi(config utils.Config) *VersionHandlerApi {
	v := new(VersionHandlerApi)
	v.xl = xlog.New("Version Handler")
	v.versionService = service2.NewVersionService(v.xl, *config.Mongo)
	return v
}

func (v *VersionHandlerApi) CreateVersion(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	var versionForm form2.VersionCreateForm
	xl.Debugf("create version")
	err := c.Bind(&versionForm)
	if err != nil {
		xl.Errorf("form binding error:%v", err)
		respErr := model.NewResponseErrorValidation(err)
		model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
		return
	}
	err = versionForm.Validate()
	if err != nil {
		xl.Errorf("form valdiation error:%v", err)
		respErr := model.NewResponseErrorValidation(err)
		model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
		return
	}
	version := model.VersionDo{
		AppName:     versionForm.AppName,
		Version:     versionForm.Version,
		Platform:    versionForm.Platform,
		CommitHash:  versionForm.CommitHash,
		UpgradeType: versionForm.UpgradeCode.Type(),
		UpgradeCode: versionForm.UpgradeCode,
		CreateAt:    time.Now(),
	}
	err = v.versionService.Create(xl, version)
	if err != nil {
		xl.Errorf("version service creating error:%v", err)
		respErr := model.NewResponseErrorValidation(err)
		model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
		return
	}
	resp := model.NewSuccessResponse("version 创建成功")
	c.JSON(http.StatusOK, resp)
}

func (v *VersionHandlerApi) GetOrListVersion(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	xl.Debugf("get version")
	versionId := c.Param("versionId")
	//userID := c.GetString(protocol.UserIDContextKey)
	pageNum := c.GetInt(model.PageNumContextKey)
	pageSize := c.GetInt(model.PageSizeContextKey)
	xl.Infof("pageNum %d pageSize %d", pageNum, pageSize)
	var filter interface{}
	var filterForm form2.VersionFilterForm
	err := c.Bind(&filterForm)
	if err != nil {
		xl.Errorf("form binding error:%v", err)
		respErr := model.NewResponseErrorValidation(err)
		model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
		return
	}
	filter = filterForm.Filter()
	xl.Debugf("filter %v", filter)
	var res interface{}
	switch versionId {
	case "":
		versions, totalCnt, err := v.versionService.GetPageByMap(v.xl, filter, pageNum, pageSize)
		if err != nil {
			xl.Errorf("version service get error:%v", err)
			respErr := model.NewResponseErrorValidation(err)
			model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
			return
		}
		endPage := totalCnt / pageSize
		if totalCnt%pageSize != 0 {
			endPage++
		}
		xl.Infof("endPage %d totalCnt mod pageNum %d totalCnt", endPage, totalCnt%pageNum)
		nextPage := pageNum + 1
		var ending bool
		if nextPage > endPage {
			ending = true
			nextPage = endPage
		}
		res = map[string]interface{}{
			"Total":          totalCnt,
			"NextId":         "",
			"Cnt":            len(versions),
			"CurrentPageNum": pageNum,
			"NextPageNum":    nextPage,
			"PageSize":       pageSize,
			"EndPage":        ending,
			"List":           versions,
		}
	default:
		version, err := v.versionService.GetOneByMap(v.xl, filter)
		if err != nil {
			xl.Errorf("version service get error:%v", err)
			respErr := model.NewResponseErrorValidation(err)
			model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
			return
		}
		res = version
	}
	resp := model.NewSuccessResponse(res)
	c.JSON(http.StatusOK, resp)
}

func (v *VersionHandlerApi) DeleteVersion(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	versionId := c.Param("versionId")
	if versionId == "" {
		xl.Errorf("no versionId found error:%v", form2.ErrVersionIdNeeded)
		respErr := model.NewResponseErrorValidation(form2.ErrVersionIdNeeded)
		model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
		return
	}
	err := v.versionService.Delete(xl, versionId)
	if err != nil {
		xl.Errorf("version service creating error:%v", err)
		respErr := model.NewResponseErrorValidation(err)
		model.NewFailResponse(*respErr).WithRequestID(xl.ReqId).Send(c)
		return
	}
	resp := model.NewSuccessResponse("version 删除成功")
	c.JSON(http.StatusOK, resp)
}
