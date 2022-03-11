package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"sync/atomic"
	"time"
)

var (
	ErrLoginFailed    = errors.New("login failed")
	ErrGetTokenFailed = errors.New("get token failed")
)

// GenerateLogUpToken 对短视频SDK日志上报进行鉴权
//
// 短视频SDK日志采用了阿里的日志服务， 日志会上报到阿里
// 移动端如果使用ak/sk上报，存在密钥泄漏的风险，因此需要移动端调用该接口拿到临时访问凭证
//
// GET /sdk/log/token
func GenerateLogUpToken(ctx *gin.Context) {
	/*
		cfg := authserver.GetCfg(ctx)
		stClient := sts.NewClient(cfg.AccessKey, cfg.AccessKeySecret, cfg.RoleArn, "")
		resp, err := stClient.AssumeRole(cfg.ExpireTime)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
			return
		}
	*/
	ctx.JSON(http.StatusOK, gin.H{})
}

type PandoraToken struct {
	ID         string
	Token      string
	StartTime  time.Time
	ExpireTime time.Time
}

// Expired 判断token是否过期
// 预留5min给客户端上传使用, 打点上报5min足够
func (t *PandoraToken) Expired() bool {
	return t.ExpireTime.Sub(time.Now()).Minutes() <= 5
}

func (t *PandoraToken) CanBeUpdated() bool {
	return t.ExpireTime.Sub(time.Now()).Minutes() <= 10
}

// TokenService 为短视频打点上报产生token
type TokenService struct {
	token *PandoraToken
	*http.Client
	*utils.PandoraConfig
	logger *xlog.Logger

	inProgress int32
}

func NewTokenService(cfg *utils.Config) *TokenService {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	return &TokenService{
		PandoraConfig: &cfg.PandoraConfig,
		Client: &http.Client{
			Jar: jar,
		},
		logger: xlog.New("token-service"),
	}
}

func (s *TokenService) Logout() error {
	resp, err := s.Client.Post(s.PandoraHost+"/api/v1/account/logout", "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ErrLoginFailed
	}

	io.Copy(ioutil.Discard, resp.Body)

	return nil
}

func (s *TokenService) Login(username, pass string) error {
	data := gin.H{
		"username": username,
		"password": pass,
	}
	bs, err := json.Marshal(data)
	if err != nil {
		return err
	}
	resp, err := s.Client.Post(s.PandoraHost+"/api/v1/account/login", "application/json", bytes.NewReader(bs))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ErrLoginFailed
	}

	io.Copy(ioutil.Discard, resp.Body)

	return nil
}

type token struct {
	ID         string `json:"id"`
	UserName   string `json:"username"`
	Audience   string `json:"audience"`
	NotBefore  int64  `json:"notBefore"`
	ExpireTime int64  `json:"expireTime"`
	IssueAt    int64  `json:"issueAt"`
	Issuer     string `json:"issuer"`
	Status     string `json:"status"`
}

type tokensResp struct {
	Tokens []token `json:"tokens"`
}

func (s *TokenService) GetTokens(username, pass string) ([]token, error) {
	if err := s.Login(username, pass); err != nil {
		return nil, err
	}
	defer s.Logout()

	resp, err := s.Client.Get(s.PandoraHost + "/api/v1/auth/tokens")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("GetTokens failed")
	}

	var tokens tokensResp
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}

	return tokens.Tokens, nil
}

func (s *TokenService) DeleteToken(tokenID, username, pass string) error {
	if err := s.Login(username, pass); err != nil {
		return err
	}
	defer s.Logout()

	req, err := http.NewRequest("DELETE", fmt.Sprintf(s.PandoraHost+"/api/v1/auth/tokens/%s", tokenID), nil)
	if err != nil {
		return err
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		s.logger.Printf("[WARN] statusCode: %d\n", resp.StatusCode)
		return fmt.Errorf("delete token failed: %s", respStr(resp.Body))
	}
	io.Copy(ioutil.Discard, resp.Body)
	return nil
}

func respStr(reader io.Reader) string {
	bs, _ := ioutil.ReadAll(reader)
	return string(bs)
}

// NewToken 获取新的token
func (s *TokenService) NewToken(username, pass string) (*PandoraToken, error) {
	if err := s.Login(username, pass); err != nil {
		return nil, err
	}
	defer s.Logout()

	now := time.Now()
	expire := now.Add(30 * time.Minute)
	bs, err := json.Marshal(gin.H{
		"username":   username,
		"audience":   "default",
		"notBefore":  now.UnixNano() / 1000000,
		"expireTime": expire.UnixNano() / 1000000,
	})
	if err != nil {
		return nil, err
	}
	resp, err := s.Client.Post(s.PandoraHost+"/api/v1/auth/tokens", "application/json", bytes.NewBuffer(bs))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get token failed: %s\n", respStr(resp.Body))
	}
	var token struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &PandoraToken{
		ID:         token.ID,
		Token:      token.Token,
		StartTime:  now,
		ExpireTime: expire,
	}, nil
}

// GetToken 获取token
func (s *TokenService) GetToken(username, pass string) (token *PandoraToken, err error) {
	if s.token != nil && s.token.CanBeUpdated() {
		if !atomic.CompareAndSwapInt32(&s.inProgress, 0, 1) {
			token = s.token
			return
		}
		go func() {
			var oldToken *PandoraToken
			for {
				s.logger.Printf("s.NewToken() in background")
				token, err = s.NewToken(username, pass)
				if err == nil {
					oldToken = s.token
					s.token = token
					break
				}
				s.logger.Printf("s.NewToken(): %v\n", err)

				time.Sleep(3)
			}
			s.inProgress = 0
			if dErr := s.DeleteToken(oldToken.ID, username, pass); dErr != nil {
				s.logger.Printf("[WARN] s.DeleteToken(): %v\n", dErr)
			}
		}()
	}
	if s.token == nil || s.token.Expired() {
		for i := 0; i < 3; i++ {
			token, err = s.NewToken(username, pass)
			if err == nil {
				s.token = token
				break
			}

			s.logger.Printf("s.NewToken(): %v\n", err)
			time.Sleep(1)
		}
	}
	token = s.token
	return
}

// PandoraUpToken GET /shortivideo/log/token
func (s *TokenService) PandoraUpToken(ctx *gin.Context) {
	token, err := s.GetToken(s.PandoraUsername, s.PandoraPass)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": token.Token})
}
