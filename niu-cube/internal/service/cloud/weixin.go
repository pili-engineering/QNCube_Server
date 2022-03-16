package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/qiniu/x/xlog"
	"github.com/solutions/niu-cube/internal/common/utils"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"
)

var (
	defaultLogger = xlog.New("default service logger")
)

type akResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

const (
	QRCodeImageInterviewPattern = "interview-%s.jpeg" // interview-<roomId>
	QRCodeImageFilePattern      = "image-file-%s"
	QRCodeImageInterviewSchema  = "pages/invite/invite"
	ErrQrCodeMsg                = "errcode"
)

type WeixinService struct {
	token   string
	baseURL string
	xl      *xlog.Logger
	conf    utils.Config
	locker  *sync.RWMutex
}

func NewWeixinService(conf utils.Config) *WeixinService {
	w := new(WeixinService)
	w.locker = &sync.RWMutex{}
	w.baseURL = "https://api.weixin.qq.com/cgi-bin"
	w.xl = xlog.New("weixin service")
	w.conf = conf
	w.setToken()
	go func() {
		ticker := time.NewTicker(time.Hour * 1)
		for {
			select {
			case <-ticker.C:
				w.setToken()
			}
		}
	}() // get new token
	return w
}

func (w *WeixinService) setToken() {
	values := url.Values{}
	values.Add("grant_type", "client_credential")
	values.Add("appid", w.conf.Weixin.AppID)
	values.Add("secret", w.conf.Weixin.AppSecret)
	res, err := http.Get(w.baseURL + "/token?" + values.Encode())
	cnt := 0
	for err != nil && cnt != 5 {
		res, err = http.Get(w.baseURL + "/token?" + values.Encode())
		cnt++
	}
	w.xl.Infof("token target url %v", w.baseURL+"/token?"+values.Encode())
	if err != nil {
		panic(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(res.Body)
	var resp akResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		w.xl.Infof("error unmarshal weixin ak body err:%v", err)
		return
	}
	w.locker.Lock()
	w.token = resp.AccessToken
	w.locker.Unlock()
	w.xl.Infof("successfully set token %v", w.token)
	return
}

func (w *WeixinService) getToken() (string, error) {
	w.locker.RLock()
	defer w.locker.RUnlock()
	if w.token == "" {
		return "", fmt.Errorf("error setting token")
	}
	return w.token, nil
}

func (w *WeixinService) getQRCode(path string) ([]byte, error) {
	payload := map[string]interface{}{
		"path":  path,
		"width": 250,
	}
	body, _ := json.Marshal(payload)
	token, err := w.getToken()
	if err != nil {
		return nil, err
	}
	target := w.baseURL + "/wxaapp/createwxaqrcode" + "?access_token=" + token
	w.xl.Infof("get qrcode,url:%s", target)
	res, err := http.Post(target, "application/json", bytes.NewReader(body))
	if err != nil {
		w.xl.Errorf("fetch qrcode failed err:%v", err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			w.xl.Errorf("close qrcode resp body failed err:%v", err)
		}
	}(res.Body)
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		w.xl.Errorf("read qrcode resp body failed err:%v", err)
		return nil, err
	}
	match, err := regexp.Match(ErrQrCodeMsg, data)
	if err != nil || match {
		w.xl.Errorf("match err reg,getting image failed")
		return nil, err
	}
	return data, err
}

func (w *WeixinService) GetAndUploadQRCode(interviewId string, interviewToken string) (string, error) {
	filekey := fmt.Sprintf(QRCodeImageInterviewPattern, interviewId)
	appletURI := QRCodeImageInterviewSchema + fmt.Sprintf("?interviewId=%s&interviewToken=%s", interviewId, interviewToken)
	image, err := w.getQRCode(appletURI)
	if err != nil {
		return "", err
	}
	w.xl.Infof("fetch qrcode of room %v successfully", interviewId)
	err = upload(w.conf.Weixin.Bucket, w.conf.QiniuKeyPair, image, filekey, w.xl)
	if err != nil {
		return "", err
	}
	imageURL := w.conf.Weixin.Link + "/" + filekey
	return imageURL, err
}

func (w *WeixinService) UploadFile(file *multipart.FileHeader) (string, error) {

	fileContent, err := file.Open()
	if err != nil {
		return "", err
	}
	defer fileContent.Close()
	fileName := fmt.Sprintf(QRCodeImageFilePattern, file.Filename)

	byteContainer, err := ioutil.ReadAll(fileContent)
	if err != nil {
		return "", err
	}
	err = upload(w.conf.Weixin.Bucket, w.conf.QiniuKeyPair, byteContainer, fileName, w.xl)
	if err != nil {
		return "", err
	}
	fileURL := w.conf.Weixin.Link + "/" + fileName
	return fileURL, err
}

// fileKey 上传文件的访问名
func upload(bucketName string, conf utils.QiniuKeyPair, data []byte, fileKey string, xl *xlog.Logger) error {
	if xl == nil {
		xl = defaultLogger
	}
	mac := qbox.NewMac(conf.AccessKey, conf.SecretKey)
	putPolicy := storage.PutPolicy{
		Scope: bucketName,
	}
	upToken := putPolicy.UploadToken(mac)
	cfg := storage.Config{}
	// 空间对应的机房
	cfg.Zone = &storage.ZoneHuanan
	// 是否使用https域名
	cfg.UseHTTPS = true
	// 上传是否使用CDN上传加速
	cfg.UseCdnDomains = false
	formUploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}
	dataLen := int64(len(data))
	err := formUploader.Put(context.Background(), &ret, upToken, fileKey, bytes.NewReader(data), dataLen, nil)
	if err != nil {
		xl.Errorf("file uploading failed err:%v", err)
		return err
	}
	xl.Infof("file upload success")
	return nil
}

func GenkodoClientToken(conf utils.QiniuKeyPair, bucket string) string {
	mac := qbox.NewMac(conf.AccessKey, conf.SecretKey)
	putPolicy := storage.PutPolicy{
		Scope: bucket,
	}
	putPolicy.Expires = 3600 * 24 * 30
	upToken := putPolicy.UploadToken(mac)
	return upToken
}
