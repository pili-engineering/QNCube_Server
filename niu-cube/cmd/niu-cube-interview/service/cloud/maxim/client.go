package maxim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	"io"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

// App Service
// 面向App的管理服务

type MaximClient struct {
	appId       string
	apiEndPoint string
	accessToken string
	client      *http.Client
}

func NewMaximClient(conf *utils.QiniuIMConfig) *MaximClient {
	return &MaximClient{
		appId:       conf.AppId,
		apiEndPoint: conf.AppEndpoint,
		accessToken: conf.AppToken,
		client:      http.DefaultClient,
	}
}

// RegisterUser /user/register/v2 注册用户,返回包含用户id
func (c *MaximClient) RegisterUser(xl *xlog.Logger, username, password string) (*gjson.Result, error) {
	if xl == nil {
		xl = xlog.New("MaximClient")
	}

	url := c.apiEndPoint + "/user/register/v2"
	var user = map[string]string{
		"username": username,
		"password": password,
	}

	resp, err := c.PostWithJson(url, user)
	defer resp.Body.Close()

	if err != nil {
		xl.Errorf("call error %+v", err)
		return nil, NewCallError(url, err)
	}

	if resp.StatusCode != 200 {
		xl.Errorf("StatusCode %d", resp.StatusCode)
		return nil, NewStatusCodeError(resp.StatusCode, "")
	}

	res, err := io.ReadAll(resp.Body)
	result := gjson.ParseBytes(res)
	if result.Get("code").Int() != 200 {
		return nil, NewMaximError(res)
	}
	return &result, nil
}

func (c *MaximClient) CreateChatroom(xl *xlog.Logger, name string) (int64, error) {
	url := c.apiEndPoint + "/group/create"
	var req = map[string]interface{}{
		"name": name,
		"type": 2,
	}
	resp, err := c.PostWithJson(url, req)
	defer resp.Body.Close()

	if err != nil {
		xl.Errorf("call error %+v", err)
		return 0, NewCallError(url, err)
	}

	if resp.StatusCode != 200 {
		xl.Errorf("StatusCode %d", resp.StatusCode)
		return 0, NewStatusCodeError(resp.StatusCode, resp.Status)
	}

	res, err := io.ReadAll(resp.Body)
	if !gjson.Valid(string(res)) {
		xl.Errorf("invalid response json %s", string(res))
		return 0, NewCallError(url, fmt.Errorf("invalid response"))
	}
	result := gjson.ParseBytes(res)
	if result.Get("code").Int() != 200 {
		return 0, NewMaximError(res)
	}

	return result.Get("data.group_id").Int(), nil
}

func (c *MaximClient) DestroyGroupChat(xl *xlog.Logger, groupId int64) error {
	query := map[string]interface{}{
		"group_id": groupId,
	}
	url := c.apiEndPoint + "/group/destroy"
	resp, err := c.PostWithEmptyBody(url, query)
	if resp == nil {
		return fmt.Errorf("maxin destroy group chat error")
	}
	defer func(body io.ReadCloser) {
		if body == nil {
			return
		}
		_ = body.Close()
	}(resp.Body)
	if err != nil {
		xl.Errorf("call error: %+v", err)
		return NewCallError(url, err)
	}
	if resp.StatusCode != 200 {
		xl.Errorf("StatusCode: %d", resp.StatusCode)
		return NewStatusCodeError(resp.StatusCode, resp.Status)
	}
	res, err := io.ReadAll(resp.Body)
	if !gjson.Valid(string(res)) {
		xl.Errorf("invalid response json %s", string(res))
		return NewCallError(url, fmt.Errorf("invalid response"))
	}
	result := gjson.ParseBytes(res)
	if result.Get("code").Int() != 200 {
		return NewMaximError(res)
	}
	if result.Get("data").Bool() {
		return nil
	} else {
		return fmt.Errorf("unknow err")
	}
}

func (c *MaximClient) PostWithJson(url string, params interface{}) (resp *http.Response, err error) {
	msg, err := json.Marshal(params)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(msg))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("app_id", c.appId)
	req.Header.Set("access-token", c.accessToken)
	req.ContentLength = int64(len(msg))

	return c.client.Do(req)
}

func (c *MaximClient) Get(url string) (*http.Response, error) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("app_id", c.appId)
	req.Header.Set("access-token", c.accessToken)
	return c.client.Do(req)
}

func (c *MaximClient) PostWithEmptyBody(url string, query map[string]interface{}) (*http.Response, error) {
	stringBuilder := strings.Builder{}
	stringBuilder.WriteString(url)
	stringBuilder.WriteString("?")
	for k, v := range query {
		stringBuilder.WriteString(k)
		stringBuilder.WriteString("=")
		stringBuilder.WriteString(fmt.Sprint(v))
		stringBuilder.WriteString("&")
	}
	str := stringBuilder.String()
	str = str[0 : len(str)-1]
	req, err := http.NewRequest("POST", str, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("app_id", c.appId)
	req.Header.Set("access-token", c.accessToken)
	return c.client.Do(req)
}
