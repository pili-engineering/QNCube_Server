package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	errors2 "github.com/solutions/niu-cube/internal/protodef/errors"
	model "github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/dao"
	"github.com/solutions/niu-cube/internal/service/db"
	"github.com/solutions/niu-cube/internal/service/web/middleware"
	"math/rand"
	"net/http"
	"regexp"
)

type SmsCodeInterface interface {
	Send(xl *xlog.Logger, phone string) (err error)
	Validate(xl *xlog.Logger, phone string, smsCode string) (err error)
}

type AccountInterface interface {
	// GetAccountByPhone 通过手机号查询账号
	GetAccountByPhone(xl *xlog.Logger, phone string) (*model.AccountDo, error)

	// GetOrSaveAccountByPhone 通过手机号查询账号或创建账号
	GetOrSaveAccountByPhone(xl *xlog.Logger, phone string) (*model.AccountDo, error)

	GetAccountByID(xl *xlog.Logger, id string) (*model.AccountDo, error)

	CreateAccount(xl *xlog.Logger, account *model.AccountDo) error

	UpdateAccount(xl *xlog.Logger, id string, account *model.AccountDo) (*model.AccountDo, error)

	AccountLogin(xl *xlog.Logger, id string) (user *model.AccountTokenDo, err error)

	AccountLogout(xl *xlog.Logger, id string) error

	DeleteAccount(xl *xlog.Logger, id string) error

	ListAll0() ([]model.AccountDo, error)
}

type AccountApiHandler struct {
	Account           AccountInterface
	SmsCode           SmsCodeInterface
	AppConfigService  db.AppConfigInterface
	DefaultAvatarURLs []string
	BaseUserDao       dao.BaseUserDaoInterface
	ExamService       ExamApi
}

// validatePhone 检查手机号码是否符合规则。
func validatePhone(phone string) bool {
	phoneRegExp := regexp.MustCompile(`1[3-9][0-9]{9}`)
	return phoneRegExp.MatchString(phone)
}

// SendSmsCode 发送验证码短信
func (h *AccountApiHandler) SendSmsCode(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	args := model.GetSmsCodeArgs{}
	err := c.Bind(&args)
	if err != nil {
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	if !validatePhone(args.Phone) {
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID).WithErrorMessage("手机号不合法")
		c.JSON(http.StatusOK, resp)
		return
	}
	messageSendErr := h.SmsCode.Send(xl, args.Phone)
	if messageSendErr != nil {
		serverErr, ok := messageSendErr.(*errors2.ServerError)
		if ok && serverErr.Code == errors2.ServerErrorSMSSendTooFrequent {
			xl.Infof("SMS code has been sent to %s, cannot resend in short time", args.Phone)
			responseErr := model.NewResponseErrorSMSSendTooFrequent()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
		xl.Errorf("failed to send sms code to phone number %s, error %v", args.Phone, messageSendErr)
		c.JSON(http.StatusInternalServerError, messageSendErr)
		return
	}
	xl.Infof("SMS code sent to number %s", args.Phone)
	h.actionLog(c).UserInfo(fmt.Sprintf("unauthorized user %s", args.Phone))
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data:    nil,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AccountApiHandler) generateNicknameByPhone(phone string) string {
	namePrefix := "用户_"
	if len(phone) < 4 {
		return namePrefix + phone
	}
	return namePrefix + phone[len(phone)-4:]
}

func (h *AccountApiHandler) generateInitialAvatar() string {
	if len(h.DefaultAvatarURLs) == 0 {
		return ""
	}
	index := rand.Intn(len(h.DefaultAvatarURLs))
	return h.DefaultAvatarURLs[index]
}

func (h *AccountApiHandler) SignUpOrIn(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	args := model.SMSLoginArgs{}
	err := c.Bind(&args)
	if err != nil {
		xl.Infof("SignUpOrIn: invalid args in body, error %v", err)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	err = h.SmsCode.Validate(xl, args.Phone, args.SMSCode)
	if err != nil {
		xl.Infof("SignUpOrIn: validate SMS code failed, error %v", err)
		responseErr := model.NewResponseErrorWrongSMSCode()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	account, err := h.Account.GetAccountByPhone(xl, args.Phone)
	if err != nil {
		if err.Error() == "not found" {
			xl.Infof("SignUpOrIn: phone number %s not found, create new account", args.Phone)
			newAccount := &model.AccountDo{
				ID:       utils.GenerateID(),
				Nickname: h.generateNicknameByPhone(args.Phone),
				Phone:    args.Phone,
				Avatar:   h.generateInitialAvatar(),
			}
			createErr := h.Account.CreateAccount(xl, newAccount)
			if createErr != nil {
				xl.Errorf("SignUpOrIn: failed to create account, error %v", err)
				responseErr := model.NewResponseErrorInternal()
				resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
				c.JSON(http.StatusOK, resp)
				return
			}
			account = newAccount
			baseUser := model.BaseUserDo{
				Id:            account.ID,
				Name:          account.Nickname,
				Nickname:      account.Nickname,
				Avatar:        account.Avatar,
				Status:        model.BaseUserLogin,
				Profile:       "",
				BaseUserAttrs: make([]model.BaseEntryDo, 0, 0),
			}
			h.BaseUserDao.Insert(nil, &baseUser)
			h.ExamService.SyncExamList(baseUser.Id)
		} else {
			xl.Errorf("SignUpOrIn: get account by phone number failed, error %v", err)
			responseErr := model.NewResponseErrorInternal()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
	}
	xl.Infof("SignUpOrIn: accountId => %s", account.ID)

	// 更新该账号状态为已登录。
	user, err := h.Account.AccountLogin(xl, account.ID)
	if err != nil {
		serverErr, ok := err.(*errors2.ServerError)
		if ok && serverErr.Code == errors2.ServerErrorUserLoggedin {
			xl.Infof("SignUpOrIn: user %s already logged in", account.ID)
			responseErr := model.NewResponseErrorAlreadyLoggedin()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
		xl.Errorf("failed to set account %s to status logged in, error %v", account.ID, err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	imUser, err := h.AppConfigService.GetUserToken(xl, user.AccountId)
	if err != nil {
		xl.Errorf("failed to call IM db to get token")
		responseErr := model.NewResponseErrorExternalService()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	h.actionLog(c).UserInfo(fmt.Sprintf("user %s", args.Phone))
	res := model.NewSuccessResponse(model.SignUpOrInResponse{
		UserInfoResponse: model.UserInfoResponse{
			ID:       account.ID,
			Nickname: account.Nickname,
			Avatar:   account.Avatar,
			Profile:  string(model.DefaultAccountProfile),
		},
		Token: user.Token,
		ImConfigResponse: model.ImConfigResponse{
			IMUsername: imUser.Username,
			IMPassword: imUser.GetPassword(),
			IMUid:      imUser.UserID,
			Type:       int(model.ImTypeQiniu),
		},
	})
	c.SetCookie(model.LoginTokenKey, user.Token, 0, "/", "niucube.qiniu.com", true, false)
	c.JSON(http.StatusOK, res)
}

func (h *AccountApiHandler) SignOut(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	userID := c.GetString(model.UserIDContextKey)
	err := h.Account.AccountLogout(xl, userID)
	if err != nil {
		xl.Errorf("user %s log out error: %v", userID, err)
		responseErr := model.NewResponseErrorNotLoggedIn()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
	}
	xl.Infof("user %s logged out", userID)
	c.SetCookie(model.LoginTokenKey, "", -1, "/", "niucube.qiniu.com", true, false)
	res := model.NewSuccessResponse(nil)
	c.JSON(http.StatusOK, res)
}

// todo signIn相同代码逻辑抽离到Service中
func (h *AccountApiHandler) SignInWithToken(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	accountId := c.GetString(model.UserIDContextKey)
	account, err := h.Account.GetAccountByID(xl, accountId)
	if err != nil {
		xl.Infof("cannot find account, accountId: %s, error %v", accountId, err)
		responseErr := model.NewResponseErrorNoSuchUser()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	// 更新该账号状态为已登录。
	user, err := h.Account.AccountLogin(xl, account.ID)
	if err != nil {
		serverErr, ok := err.(*errors2.ServerError)
		if ok && serverErr.Code == errors2.ServerErrorUserLoggedin {
			xl.Infof("SignUpOrIn: user %s already logged in", account.ID)
			responseErr := model.NewResponseErrorAlreadyLoggedin()
			resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
			c.JSON(http.StatusOK, resp)
			return
		}
		xl.Errorf("failed to set account %s to status logged in, error %v", account.ID, err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	// TODO 融云获取需要用户ID，不同设备，相同七牛账号如何处理，是否可以用UUID
	imUser, err := h.AppConfigService.GetUserToken(xl, user.AccountId)
	if err != nil {
		xl.Errorf("failed to call IM db to get token")
		responseErr := model.NewResponseErrorExternalService()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	res := model.NewSuccessResponse(model.SignUpOrInResponse{
		UserInfoResponse: model.UserInfoResponse{
			ID:       account.ID,
			Nickname: account.Nickname,
			Avatar:   account.Avatar,
			Profile:  string(model.DefaultAccountProfile),
		},
		Token: user.Token,
		ImConfigResponse: model.ImConfigResponse{
			IMUsername: imUser.Username,
			IMPassword: imUser.GetPassword(),
			IMUid:      imUser.UserID,
			Type:       int(model.ImTypeQiniu),
		},
	})
	c.SetCookie(model.LoginTokenKey, user.Token, 0, "/", "niucube.qiniu.com", true, false)
	c.JSON(http.StatusOK, res)
}

func (h *AccountApiHandler) UpdateAccountInfo(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	accountId := c.GetString(model.UserIDContextKey)

	args := model.UpdateAccountInfoArgs{}
	bindErr := c.Bind(&args)
	if bindErr != nil {
		xl.Infof("invalid args in request body, error %v", bindErr)
		responseErr := model.NewResponseErrorBadRequest()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	account, err := h.Account.GetAccountByID(xl, accountId)
	if err != nil {
		xl.Infof("cannot find account, accountId: %s, error %v", accountId, err)
		responseErr := model.NewResponseErrorNoSuchUser()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	if account.ID != "" && account.ID != accountId {
		xl.Infof("user %s try to update profile of other user %s", accountId, account.ID)
		responseErr := model.NewResponseErrorUnauthorized()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	if args.Nickname != "" {
		account.Nickname = args.Nickname
	}
	if args.Avatar != "" {
		account.Avatar = args.Avatar
	}

	newAccount, err := h.Account.UpdateAccount(xl, accountId, account)
	if err != nil {
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	res := model.NewSuccessResponse(model.UpdateAccountInfoResponse{
		UserInfoResponse: model.UserInfoResponse{
			ID:       newAccount.ID,
			Nickname: newAccount.Nickname,
			Avatar:   newAccount.Avatar,
			Phone:    newAccount.Phone,
			Profile:  string(model.DefaultAccountProfile),
		},
	})
	c.JSON(http.StatusOK, res)
}

func (h *AccountApiHandler) GetAccountInfo(c *gin.Context) {
	xl := c.MustGet(model.XLogKey).(*xlog.Logger)
	requestID := xl.ReqId
	accountId := c.GetString(model.UserIDContextKey)
	account, err := h.Account.GetAccountByID(xl, accountId)
	if err != nil {
		xl.Infof("cannot find account, error %v", err)
		responseErr := model.NewResponseErrorNoSuchUser()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}
	if account.ID != "" && account.ID != accountId {
		xl.Infof("user %s try to get account info of other user %s", accountId, account.ID)
		responseErr := model.NewResponseErrorUnauthorized()
		resp := model.NewFailResponse(*responseErr).WithRequestID(requestID)
		c.JSON(http.StatusOK, resp)
		return
	}

	res := model.NewSuccessResponse(model.GetAccountInfoResponse{
		UserInfoResponse: model.UserInfoResponse{
			ID:       account.ID,
			Nickname: account.Nickname,
			Avatar:   account.Avatar,
			Phone:    account.Phone,
			Profile:  string(model.DefaultAccountProfile),
		},
	})
	c.JSON(http.StatusOK, res)
}

func (h *AccountApiHandler) Sync(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	accountDos, err := h.Account.ListAll0()
	if err != nil {
		xl := context.MustGet(model.XLogKey).(*xlog.Logger)
		xl.Errorf("cannot list all accounts, error %v", err)
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr)
		context.JSON(http.StatusOK, resp)
		return
	}
	for i := range accountDos {
		user, _ := h.BaseUserDao.Select(nil, accountDos[i].ID)
		if user == nil {
			baseUser := model.BaseUserDo{
				Id:            accountDos[i].ID,
				Name:          accountDos[i].Nickname,
				Nickname:      accountDos[i].Nickname,
				Avatar:        accountDos[i].Avatar,
				Status:        model.BaseUserLogin,
				Profile:       "",
				BaseUserAttrs: make([]model.BaseEntryDo, 0, 0),
			}
			h.BaseUserDao.Insert(nil, &baseUser)
		}
		xl.Infof("sync: %s", accountDos[i].Phone)
		h.ExamService.SyncExamList(accountDos[i].ID)
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
		}{},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (h *AccountApiHandler) DeleteAccount(context *gin.Context) {
	xl := context.MustGet(model.XLogKey).(*xlog.Logger)
	requestId := xl.ReqId
	userId := context.GetString(model.UserIDContextKey)
	phone := context.Param("phone")
	xl.Infof("user: %s try to delete %s.", userId, phone)
	accountDo, err := h.Account.GetAccountByPhone(nil, phone)
	if err != nil {
		responseErr := model.NewResponseErrorNoSuchUser()
		resp := model.NewFailResponse(*responseErr)
		context.JSON(http.StatusOK, resp)
		return
	}
	err = h.Account.DeleteAccount(nil, accountDo.ID)
	if err != nil {
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr)
		context.JSON(http.StatusOK, resp)
		return
	}
	err = h.BaseUserDao.Delete(nil, accountDo.ID)
	if err != nil {
		responseErr := model.NewResponseErrorInternal()
		resp := model.NewFailResponse(*responseErr)
		context.JSON(http.StatusOK, resp)
		return
	}
	resp := &model.Response{
		Code:    int(model.ResponseStatusCodeSuccess),
		Message: string(model.ResponseStatusMessageSuccess),
		Data: struct {
		}{},
		RequestID: requestId,
	}
	context.JSON(http.StatusOK, resp)
}

func (h *AccountApiHandler) actionLog(c *gin.Context) *middleware.Action {
	val, ok := c.Get(model.ActionLogContentKey)
	if ok {
		return val.(*middleware.Action)
	} else {
		a := &middleware.Action{}
		c.Set(model.ActionLogContentKey, a)
		return a
	}
}
