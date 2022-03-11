package middleware

import (
	"encoding/base64"
	"encoding/json"
	model "github.com/solutions/niu-cube/internal/protodef/model"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
)

// Authenticate 校验请求者的身份。
func Authenticate(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	// TODO 获取UA信息
	FetchUserAgent(xl, requestID, c)

	// 优先根据Authorization:Bearer <token>校验。
	FetchTokenFromHeader(xl, requestID, c)
}

func AfapAuthenticate(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	// 无登录状态用户根据interviewToken进行获取用户ID
	interviewToken := c.PostForm("interviewToken")
	val, err := url.QueryUnescape(interviewToken)
	if err != nil {
		interviewToken = val
	}

	// TODO 获取UA信息
	FetchUserAgent(xl, requestID, c)

	if interviewToken == "" {
		xl.Debugf("no interviewToken in POST form.")
		interviewToken = c.Query("interviewToken")
	}
	if interviewToken != "" {
		xl.Debugf("fetch interviewToken success. inteviewToken: %s", interviewToken)
		decodedInterviewParams, err := base64.StdEncoding.DecodeString(interviewToken)
		if err == nil {
			interviewTokenArgs := model.InterviewTokenArgs{}
			err = json.Unmarshal([]byte(decodedInterviewParams), &interviewTokenArgs)
			if err == nil && interviewTokenArgs.UserID != "" {
				userID := interviewTokenArgs.UserID
				user, _ := accountService.GetAccountByID(xl, interviewTokenArgs.UserID)
				c.Set(model.UserContextKey, *user)
				c.Set(model.UserIDContextKey, userID)
				c.Set(model.TokenSourceContextKey, model.TokenSourceFromInterviewToken)
				xl.Debugf("fetch interviewToken success. inteviewToken: %s, userID: %s", interviewToken, userID)
			} else {
				xl.Debugf("fetch interviewToken fail. inteviewToken: %s", interviewToken)
			}
		} else {
			xl.Debugf("Decode interviewToken fail. inteviewToken: %s", interviewToken)
		}
	}

	_, exist := c.Get(model.UserIDContextKey)
	if !exist {
		// 根据Authorization:Bearer <token>校验。
		FetchTokenFromHeader(xl, requestID, c)
	}

}

func FetchTokenFromHeader(xl *xlog.Logger, requestID string, c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		xl.Debug("authorization header is empty or in wrong format")
		xl.Debugf("auth header: %v", authHeader)
		xl.Debugf("%s %s: request unauthorized, wrong auth header format", c.Request.Method, c.Request.URL.Path)

		responseErr := model.NewResponseErrorNotLoggedIn()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		c.Abort()
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	id, err := accountService.GetIDByToken(xl, token)

	if err != nil {
		xl.Debugf("%s %s: request unauthorized, error %v", c.Request.Method, c.Request.URL.Path, err)
		responseErr := model.NewResponseErrorBadToken()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		c.Abort()
		return
	}
	user, _ := accountService.GetAccountByID(xl, id)
	c.Set(model.UserContextKey, *user)
	c.Set(model.UserIDContextKey, id)
	c.Set(model.TokenSourceContextKey, model.TokenSourceFromHeader)
}

func FetchUserAgent(xl *xlog.Logger, requestID string, c *gin.Context) {
	uaHeader := c.GetHeader("User-Agent")
	mobileKeywords := []string{"Silk/", "Kindle",
		"BlackBerry", "Opera Mini", "Opera Mobi"}
	mobileKeywords4Android := []string{"Android"}
	mobileKeywords4Apple := []string{"Mobile"}

	for i := 0; i < len(mobileKeywords4Android); i++ {
		if strings.Contains(uaHeader, mobileKeywords4Android[i]) {
			c.Set(model.UAContextKey, model.UAMobileAndroid)
			break
		}
	}
	for i := 0; i < len(mobileKeywords4Apple); i++ {
		if strings.Contains(uaHeader, mobileKeywords4Apple[i]) {
			c.Set(model.UAContextKey, model.UAMobileApple)
			break
		}
	}
	for i := 0; i < len(mobileKeywords); i++ {
		if strings.Contains(uaHeader, mobileKeywords[i]) {
			c.Set(model.UAContextKey, model.UAMobile)
			break
		}
	}
}
