package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/protodef/model"
	"math/rand"
	"strconv"
)

func MockAuth(c *gin.Context) {
	user := RandomUser()
	candidate := RandomUser()
	c.Set(protodef.ContextUserKey, user)
	c.Set(protodef.ContextUserIdKey, user.ID)

	c.Set(protodef.ContextCandidateKey, candidate)
	c.Set(protodef.ContextCandidateIdKey, candidate.ID)
}

func RandomUser() model.Account {
	userId := strconv.Itoa(rand.Int() % 2000)
	userPhone := ""
	for i := 0; i < 11; i++ {
		userPhone += strconv.Itoa(rand.Intn(10))
	}
	user := model.Account{
		ID:       userId,
		Phone:    userPhone,
		Nickname: "user" + userId,
	}
	return user
}
