package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/cloud"
	dao2 "github.com/solutions/niu-cube/internal/service/dao"
	"net/http"
)

type FileApiHandler struct {
	weixin       *cloud.WeixinService
	imageFileDao dao2.ImageFileDaoInterface
}

func NewFileApiHandler(conf utils.Config) *FileApiHandler {
	xl := xlog.New("file-api")
	i := new(FileApiHandler)
	//i.weixin = cloud.NewWeixinService(conf)

	imageFileDao, err := dao2.NewImageFileDao(xl, conf.Mongo)
	if err != nil {
		xl.Error("create ImageFileDao failed.")
		return nil
	}
	i.imageFileDao = imageFileDao

	return i
}

func (h *FileApiHandler) Upload(c *gin.Context) {

	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	file, err := c.FormFile("file")

	if err != nil {
		xl.Errorf("failed to Upload, error %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	fileName := file.Filename
	xl.Info("fileName is:", fileName)
	url, err := h.weixin.UploadFile(file)
	if err != nil {
		xl.Errorf("failed to Upload, error %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	imageFileDo := &model.ImageFileDo{
		FileName: fileName,
		FileUrl:  url,
		Status:   model.ImageFileStatusNormal,
	}
	imageFile, err := h.imageFileDao.InsertImageFile(xl, imageFileDo)
	if err != nil {
		xl.Errorf("InsertImageFile, error %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      imageFile,
		RequestID: requestID,
	}
	c.JSON(http.StatusOK, resp)
	return
}

func (h *FileApiHandler) RecentImage(c *gin.Context) {

	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId

	imageFile, err := h.imageFileDao.SelectRecentImage(xl)

	if err != nil {
		xl.Errorf("SelectRecentImage, error %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	resp := &model.Response{
		Code:      int(model.ResponseStatusCodeSuccess),
		Message:   string(model.ResponseStatusMessageSuccess),
		Data:      imageFile,
		RequestID: requestID,
	}
	c.JSON(http.StatusOK, resp)
	return

}
