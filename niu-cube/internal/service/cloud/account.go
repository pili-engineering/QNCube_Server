package cloud

import (
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/solutions/niu-cube/internal/common/utils"
)

func GetToken(conf utils.QiniuKeyPair, src string) string {
	mac := qbox.NewMac(conf.AccessKey, conf.SecretKey)
	token := mac.Sign([]byte(src))
	return token
}
