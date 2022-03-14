package handler

import (
	model "github.com/solutions/niu-cube/internal/protodef/model"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
)

type PageHandlerApi struct {
}

func (h *PageHandlerApi) FetchPageInfo(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	pageNumArg := c.DefaultQuery("pageNum", "1")
	pageSizeArg := c.DefaultQuery("pageSize", "10")
	pageNum, err := strconv.Atoi(pageNumArg)
	if err != nil {
		xl.Infof("FetchPageInfo.pageNum transfer int err, use default value %v", err)
		pageNum = 1
	}
	pageSize, err := strconv.Atoi(pageSizeArg)
	if err != nil {
		xl.Infof("FetchPageInfo.pageSize transfer int err, use default value %v", err)
		pageSize = 10
	}
	c.Set(model.PageNumContextKey, pageNum)
	c.Set(model.PageSizeContextKey, pageSize)
}
