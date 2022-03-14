package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef"
	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
)

func SetUpReq(c *gin.Context) {
	var requestID string
	if requestID = c.Request.Header.Get(protodef.RequestIDHeader); requestID == "" {
		requestID = utils.NewReqID()
		c.Request.Header.Set(model.RequestIDHeader, requestID)
	}
	xl := xlog.New(requestID)
	xl.Debugf("request: %s %s", c.Request.Method, c.Request.URL.Path)
	c.Set(protodef.XLogKey, xl)
	//c.Set(model.RequestStartKey, time.Now())
}
