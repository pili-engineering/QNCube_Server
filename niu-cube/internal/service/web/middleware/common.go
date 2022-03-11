package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/db/dao"
	"gopkg.in/mgo.v2"
	"log"
	"strconv"
	"strings"
	"time"
)

var (
	defaultActionManager = NewActionManager(nil)
)

func FetchPageInfo(c *gin.Context) {
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

func ActionLogMiddleware() gin.HandlerFunc {
	// all action divide into 2 kind
	// before Login 、 after Login
	return func(c *gin.Context) {
		path := c.FullPath()
		method := c.Request.Method
		actionLog, ok := defaultActionManager.MatchRoute(method, path)
		c.Set(model.ActionLogContentKey, actionLog)
		c.Next()
		val, _ := c.Get(model.ActionLogContentKey)
		fromContext := val.(*Action)
		log.Printf("match: %v log: %v", ok, fromContext.With(c))
		record := fromContext.genRecord()
		defaultActionManager.Save(record)
	}
}

var methodMsg = map[string]string{
	"POST":   "创建",
	"GET":    "获取",
	"DELETE": "删除",
	"PUT":    "更新",
}

var routeMsg = map[string]string{
	"GET accountInfo": "账户信息",

	"POST interview": "面试",
	"GET interview":  "面试详情",

	"heartBeat":       "传了面试心跳",
	"getSmsCode":      "获取验证码",
	"signUpOrIn":      "登入",
	"signOut":         "登出",
	"signInWithToken": "Token登录",
	"cancelInterview": "取消面试",
	"endInterview":    "结束面试",
	"joinInterview":   "加入面试",
	"leaveInterview":  "离开面试",
	"token getToken":  "get token",
}

type Action struct {
	method    string
	subject   string
	userInfo  string
	userPhone string
	msg       string
	time      time.Time
}

func NewAction(method string, subject string, msg string) *Action {
	return &Action{method: method, subject: subject, msg: msg}
}

type ActionRecord struct {
	Msg       string    `json:"msg"`
	UserPhone string    `json:"user_phone"`
	Time      time.Time `json:"time"`
	Method    string    `json:"method"`
	Subject   string    `json:"subject"`
}

func NewActionRecord(msg string, userPhone string, method string, subject string) *ActionRecord {
	return &ActionRecord{Msg: msg, UserPhone: userPhone, Time: time.Now(), Method: method, Subject: subject}
}

type ActionManager struct {
	Actions    []*Action
	actionColl *mgo.Collection
	xl         *xlog.Logger
}

func NewActionManager(actions map[Action]string) *ActionManager {
	am := &ActionManager{xl: xlog.New("middleware.ActionManager")}
	if actions == nil {
		defaultActions := make([]*Action, 0)
		for k, v := range routeMsg {
			method, subject := parseMethodAndSubject(k)
			action := NewAction(method, subject, v)
			defaultActions = append(defaultActions, action)
		}
		am.Actions = defaultActions
	}
	return am
}

func (am *ActionManager) MatchRoute(method, path string) (*Action, bool) {
	pathSpec := parsePath(path)
	subject := strings.Join(pathSpec, " ")
	for _, action := range am.Actions {
		if (action.method == "ALL" || action.method == method) && action.subject == subject {
			return action, true
		}
	}
	return NewAction(method, subject, "default"), true
}

func (am *ActionManager) Save(action *ActionRecord) {
	if am.actionColl == nil {
		client, err := mgo.Dial(utils.DefaultConf.Mongo.URI)
		if err != nil {
			am.xl.Fatalf("err connect mongo:%s", err)
		} else {
			am.actionColl = client.DB(utils.DefaultConf.Mongo.Database).C(dao.ActionCollection)
		}
	}
	err := am.actionColl.Insert(action)
	if err != nil {
		xl.Errorf("failed save action:%s", action)
	}
}

func (a Action) String() string {
	methodStr := ""
	if a.method != "ALL" {
		methodStr += methodMsg[a.method]
	}
	return fmt.Sprintf("%s %s%s", a.userInfo, methodStr, a.msg)
}

func (a *Action) With(c *gin.Context) Action {
	val, ok := c.Get(model.UserContextKey)
	if !ok {
		return *a
	}
	user, ok := val.(model.AccountDo)
	if !ok {
		return *a
	}
	a.userInfo = fmt.Sprintf("user %s", user.Phone)
	a.userPhone = user.Phone
	return *a
}

func (a *Action) UserInfo(info string) {
	a.userInfo = info
}

func (a *Action) Msg(msg string) {
	a.msg = msg
}

func (a *Action) genRecord() *ActionRecord {
	r := NewActionRecord(a.String(), a.userPhone, a.method, a.subject)
	return r
}

// /v1/leaveInterview/:interviewId -> leaveInterview
// /v1/ie/:roomId/create 0> ie create
// parsePath skip first path item && skip param,may return nil
func parsePath(path string) []string {
	fields := strings.Split(path, "/")
	if len(fields) < 2 {
		return nil
	}
	noVersionFields := fields[2:]
	res := make([]string, 0)
	for _, part := range noVersionFields {
		if !strings.HasPrefix(part, ":") {
			res = append(res, part)
		}
	}
	return res
}

// GET interview -> method="GET" subject="interview"
// signInWithToken -> method="ALL" subject="signInWithToken"
func parseMethodAndSubject(val string) (method, subject string) {
	val = strings.TrimSpace(val)
	if val == "" {
		return "", ""
	} else {
		allowMethods := []string{"GET", "POST", "PUT", "DELETE"}
		for _, m := range allowMethods {
			index := strings.Index(val, m)
			if index != -1 {
				return m, strings.TrimSpace(val[index+len(m):])
			}
		}
		return "ALL", val
	}
}
