package utils

import (
	"math/rand"
	"strings"
	"time"
)

func IsFixedPhone(phone string) bool {
	if _, ok := DefaultConf.SMS.FixedCodes[phone]; ok {
		return true
	}
	return false
}

const AlphaNum = "0123456789abcdefghijklmnopqrstuvwxyz"

// GenerateID utils func: for 12-digit random id generation
func GenerateID() string {
	idLength := 12
	stringBuilder := strings.Builder{}
	for i := 0; i < idLength; i++ {
		index := rand.Intn(36)
		stringBuilder.WriteRune(rune(AlphaNum[index]))
	}
	return stringBuilder.String()
}

func TimedTask(t time.Time, task func()) {
	if t.Before(time.Now()) {
		go task()
	} else {
		go func() {
			<-time.After(time.Duration(t.UnixMilli()-time.Now().UnixMilli()) * time.Millisecond)
			task()
		}()
	}
}
