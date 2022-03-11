package main

import (
	"github.com/gin-gonic/gin"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/common"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/service/cloud"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/service/db"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/service/handler"
	"github.com/solutions/niu-cube/cmd/niu-cube-interview/service/middleware"
)

func main() {
	router := gin.Default()
	roomService := cloud.NewRoomService()
	imService := cloud.NewRongCloudIMService()
	interviewService := db.NewInterviewService(roomService)
	accountService := db.NewAccountService()

	interviewHandler := handler.NewInterviewHandle(interviewService)
	accountHandler := handler.NewAccountHandler(accountService, imService)

	v2 := router.Group("v2", middleware.SetUpReq)
	// interview route
	{
		interview := v2.Group("", middleware.Authenticate(), middleware.MockAuth)
		interviewHandler.RegisterRoute(interview)
	}
	// account route
	{
		account := v2.Group("")
		accountHandler.RegisterRoute(account)
	}
	//

	router.Run(common.GetConf().ListenAddr)
}
