package common

import (
	"encoding/base64"
	"encoding/binary"
	"github.com/dgrijalva/jwt-go"
	"math/rand"
	"time"
)

// GenerateID utils func: for 12-digit random id generation
func GenerateID() string {
	alphaNum := "0123456789abcdefghijklmnopqrstuvwxyz"
	idLength := 12
	id := ""
	for i := 0; i < idLength; i++ {
		index := rand.Intn(len(alphaNum))
		id = id + string(alphaNum[index])
	}
	return id
}

// IsFixedPhone whethre phone is in fixed phone list
func IsFixedPhone(phone string) bool {
	if _, ok := GetConf().SMS.FixedCodes[phone]; ok {
		return true
	}
	return false
}

var pid = uint32(time.Now().UnixNano() % 4294967291)

// NewReqID for generate req id
func NewReqID() string {
	var b [12]byte
	binary.LittleEndian.PutUint32(b[:], pid)
	binary.LittleEndian.PutUint64(b[4:], uint64(time.Now().UnixNano()))
	return base64.URLEncoding.EncodeToString(b[:])
}

// JwtSign sign map[string]interface{} data,return signed string
func JwtSign(data interface{}) string {
	container, ok := data.(map[string]interface{})
	if !ok {
		panic("jwt sign nead map[string]interface{} payload")
	}
	claims := jwt.MapClaims(container)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedString, err := token.SignedString([]byte(GetConf().JwtKey))
	if err != nil {
		panic(err)
	}
	return signedString
}

// JwtDecode will not validate exp,just decode with sha256 + key in conf
func JwtDecode(token string) (map[string]interface{}, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(GetConf().JwtKey), nil
	})
	if err != nil {
		return nil, err
	} else {
		return claims, err
	}
}
