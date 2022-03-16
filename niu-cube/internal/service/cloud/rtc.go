package cloud

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io/ioutil"
	"net/http"
	"time"

	qiniuauth "github.com/qiniu/go-sdk/v7/auth"
	qiniurtc "github.com/qiniu/go-sdk/v7/rtc"
	"github.com/qiniu/x/xlog"

	"github.com/solutions/niu-cube/internal/common/utils"
)

type RTCService struct {
	*qiniurtc.Manager
	conf   utils.QiniuRTCConfig
	signer *qiniuauth.Credentials
	xl     *xlog.Logger
}

const (
	SDKInvokeTimeout = time.Second * 5
	// DefaultRTCRoomTokenTimeout 默认的RTC加入房间用token的过期时间。
	DefaultRTCRoomTokenTimeout = 60 * time.Second
	FLvSuffix                  = "flv"
	M3u8Suffix                 = "m3u8"
)

func NewRtcService(conf utils.Config) *RTCService {
	r := new(RTCService)
	r.conf = *conf.RTC
	r.xl = xlog.New("rtc db")
	r.signer = &qiniuauth.Credentials{
		AccessKey: conf.QiniuKeyPair.AccessKey,
		SecretKey: []byte(conf.QiniuKeyPair.SecretKey),
	}
	client := qiniurtc.NewManager(r.signer)
	r.Manager = client
	return r
}

func (r *RTCService) ListUser(roomId string) (res []string, err error) {
	users, err := r.Manager.ListUser(r.conf.AppID, roomId)
	color.Blue(fmt.Sprintf("%d", len(users)))
	if err != nil {
		return nil, err
	} else {
		res = make([]string, 0, len(users))
		for _, u := range users {
			res = append(res, u.UserID)
		}
		return
	}
}

func (r *RTCService) KickUser(roomId, userId string) error {
	return r.Manager.KickUser(r.conf.AppID, roomId, userId)
}

func (r *RTCService) Online(roomId, userId string) bool {
	result := make(chan bool)
	go func() {
		users, err := r.ListUser(roomId)
		if err != nil {
			result <- false
		}
		for _, id := range users {
			if id == userId {
				result <- true
			}
		}
		result <- false
	}()
	select {
	case res := <-result:
		return res
	case <-time.After(SDKInvokeTimeout):
		r.xl.Infof("rtc db list users timeout")
		return false
	}
}

func (r *RTCService) GenerateRTCRoomToken(roomId, userId, permission string) string {
	roomTimeOut := DefaultRTCRoomTokenTimeout
	if r.conf.RoomTokenExpireSecond > 0 {
		roomTimeOut = time.Duration(r.conf.RoomTokenExpireSecond) * time.Second
	}
	roomAccess := qiniurtc.RoomAccess{
		AppID:      r.conf.AppID,
		RoomName:   roomId,
		UserID:     userId,
		ExpireAt:   time.Now().Add(roomTimeOut).Unix(),
		Permission: permission,
	}
	token, _ := r.GetRoomToken(roomAccess)
	return token
}

func (r *RTCService) RecordPlayBackM3u8(streamName string, from, to int64, callback func(filename map[string]string, ok bool) error) error {
	encodedStreamName := base64.StdEncoding.EncodeToString([]byte(streamName))
	params := map[string]interface{}{
		"fname":  streamName,
		"start":  from,
		"end":    to,
		"format": "m3u8",
	}
	val, _ := json.Marshal(params)
	url := fmt.Sprintf("https://pili.qiniuapi.com/v2/hubs/%s/streams/%s/saveas", r.conf.Hub, encodedStreamName)
	req, err := http.NewRequest("POST", url, bytes.NewReader(val))
	if err != nil {
		r.xl.Errorf("error making req err:%v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	sign, err := r.signer.SignRequestV2(req)
	if err != nil {
		r.xl.Errorf("error signing req err:%v", err)
		return err
	}
	req.Header.Set("Authorization", "Qiniu "+sign)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		r.xl.Errorf("error invoke api err:%v", err)
		return err
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		r.xl.Errorf("error read body err:%v", err)
		return err
	}
	resp := make(map[string]string, 0)
	_ = json.Unmarshal(data, &resp)
	_, ok := resp["fname"]
	err = callback(resp, ok)
	if err != nil {
		return err
	}
	return nil
}

func (r *RTCService) CreateMerge(interviewId string, interviewerId string, otherIds ...string) error {
	streamName := r.streamName(interviewId)
	users := []mergeUser{
		{
			UserID:      interviewerId,
			StretchMode: "aspectFill",
			Sequence:    1,
		},
	}
	for i, u := range otherIds {
		guest := mergeUser{
			UserID:      u,
			StretchMode: "aspectFill",
			Sequence:    i + 2,
		}
		users = append(users, guest)
	}
	args := createMergeJobArgs{
		ID:            "job-" + interviewId,
		AudioOnly:     false,
		Width:         640,
		Height:        480,
		Fps:           25,
		Kbps:          1000,
		MinRateKbps:   1000,
		MaxRateKbps:   1000,
		StretchMode:   "aspectFill",
		PublishURL:    fmt.Sprintf("rtmp://pili-publish.qnsdk.com/%s/%s", r.conf.Hub, streamName),
		Background:    background{},
		HoldLastFrame: false,
		Template:      "horizontal",
		UserInfos:     users,
	}
	url := fmt.Sprintf("https://rtc.qiniuapi.com/v3/apps/%s/rooms/%s/merge", r.conf.AppID, interviewId)
	val, _ := json.Marshal(args)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(val))
	req.Header.Set("Content-Type", "application/json")
	sign, _ := r.signer.SignRequestV2(req)
	req.Header.Set("Authorization", "Qiniu "+sign)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		r.xl.Errorf("error invoke api %s:%v", url, err)
		return err
	}
	resp := make(map[string]string, 0)
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		r.xl.Errorf("error unmarshal err:%v", err)
		return err
	}
	r.xl.Infof("request %v", req)
	r.xl.Infof("success create merge job %v", resp)
	return nil
}

func (r *RTCService) streamName(interviewId string) string {
	return fmt.Sprintf(r.conf.StreamPattern, interviewId)
}

func (r *RTCService) StreamPubURL(interviewId string) string {
	return fmt.Sprintf(r.conf.PublishURL + "/" + r.conf.Hub + "/" + r.streamName(interviewId))
}

func (r *RTCService) StreamRtmpPlayURL(interviewId string) string {
	return fmt.Sprintf(r.conf.RtmpPlayURL + "/" + r.conf.Hub + "/" + r.streamName(interviewId))
}

func (r *RTCService) StreamFlvPlayURL(interviewId string) string {
	return fmt.Sprintf(r.conf.FlvPlayURL + "/" + r.conf.Hub + "/" + r.streamName(interviewId) + "." + FLvSuffix)
}

func (r *RTCService) StreamHlsPlayURL(interviewId string) string {
	return fmt.Sprintf(r.conf.HlsPlayURL + "/" + r.conf.Hub + "/" + r.streamName(interviewId) + "." + M3u8Suffix)
}

type createMergeJobArgs struct {
	ID            string      `json:"RoomId"`
	AudioOnly     bool        `json:"audioOnly"`
	Width         int         `json:"width"`
	Height        int         `json:"height"`
	Fps           int         `json:"fps"`
	Kbps          int         `json:"kbps"`
	MinRateKbps   int         `json:"minRate"`
	MaxRateKbps   int         `json:"maxRate"`
	StretchMode   string      `json:"stretchMode"`
	PublishURL    string      `json:"publishUrl"`
	Background    background  `json:"background"`
	Watermarks    []watermark `json:"watermarks"`
	HoldLastFrame bool        `json:"holdLastFrame"`
	Template      string      `json:"template"`
	UserInfos     []mergeUser `json:"userInfos"`
}
type background struct {
	URL         string `json:"url"`
	Width       int    `json:"w"`
	Height      int    `json:"h"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	StretchMode string `json:"stretchMode"`
}

type watermark struct {
	URL         string `json:"url"`
	Width       int    `json:"w"`
	Height      int    `json:"h"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	StretchMode string `json:"stretchMode"`
}

type mergeUser struct {
	UserID        string `json:"userId"`
	BackgroundURL string `json:"backgroundUrl"`
	StretchMode   string `json:"stretchMode"`
	Sequence      int    `json:"sequence"`
}
