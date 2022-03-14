package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/common"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef"
	"net/http"
	"strings"
	"time"
)

var (
	defaultLogger = xlog.New("middleware")
)

func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		bearTokenString := c.GetHeader(protodef.HeaderTokenKey)
		tokenString := strings.TrimPrefix(bearTokenString, "Bearer ")
		claims, err := common.JwtDecode(tokenString)
		switch {
		case err != nil:
			defaultLogger.Errorf("error decode jwt token:%v", err)
			c.JSON(http.StatusUnauthorized, protodef.UnAuthorizedResponse)
			c.Abort()
		default:
			id, ok := claims[protodef.ContextUserIdKey]
			// not userId in jwt-token
			if !ok {
				defaultLogger.Errorf("error retrieve userId in jwt-token err:%v", err)
				c.JSON(http.StatusUnauthorized, protodef.UnAuthorizedResponse)
				c.Abort()
			} else {
				// retrieve userId from jwt-token and set in context
				defaultLogger.Infof("user %v authed at %v", id, time.Now())
				c.Set(protodef.ContextUserIdKey, id)
				c.Next()
			}
		}
	}
}
