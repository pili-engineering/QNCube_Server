package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	model "github.com/solutions/niu-cube/internal/protodef/model"
	service "github.com/solutions/niu-cube/internal/service/db"
	"net/http"
)

var (
	versionService *service.VersionService
	accountService *service.AccountService
	xl             = xlog.New("Middleware")
)

func InitMiddleware(conf utils.Config) {
	var err error
	versionService = service.NewVersionService(xl, *conf.Mongo)
	accountService, err = service.NewAccountService(*conf.Mongo, xl)
	if err != nil {
		xl.Fatalf("error creating account service err:%v", err)
	}
	return
}

func VersionGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetString(model.UserIDContextKey)
		user, err := accountService.GetAccountByID(xl, userId)
		if err != nil {
			xl.Infof("user %v not exists err:%v", userId, err)
			// TODO: 文档记录 使用goto替代重复代码
			resp := model.NewResponseErrorUnauthorized()
			c.JSON(http.StatusUnauthorized, resp)
			c.Abort()
			return
		}
		if _, ok := utils.DefaultConf.SMS.FixedCodes[user.Phone]; !ok {
			xl.Infof("user %v with phone %v permission denied ", userId, user.Phone)
			// TODO: 文档记录
			resp := model.NewResponseErrorUnauthorized()
			c.JSON(http.StatusUnauthorized, resp)
			c.Abort()
			return
		}
		c.Next()
	}
}
