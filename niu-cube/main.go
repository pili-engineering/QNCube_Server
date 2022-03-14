// Copyright 2020 Qiniu Cloud (qiniu.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// @title 互动直播API
// @version 1.0
// @description 互动直播API
// @termsOfService https://www.qiniu.com

// @contact.name niu cube developer
// @contact.url https://github.com/qrtc/qlive
// @contact.email

// @license.name Apache 2.0
// @license.url https://www.apache.org/licenses/LICENSE-2.0

// @host localhost:8080
// @BasePath /v1

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/service/task"
	"github.com/solutions/niu-cube/internal/service/web"

	"github.com/jasonlvhit/gocron"
	"github.com/qiniu/x/log"
)

var (
	configFilePath = "niu-cube.conf"
)

func main() {
	fmt.Println(time.Now())
	flag.StringVar(&configFilePath, "f", configFilePath, "configuration file to run niu-cube server")
	flag.Parse()

	utils.InitConf(configFilePath)
	log.SetOutputLevel(utils.DefaultConf.DebugLevel)
	rand.Seed(time.Now().UnixNano())
	// 启动定时任务
	go func() {
		interviewTask, _ := task.NewInterviewTask(utils.DefaultConf.Mongo.URI, utils.DefaultConf.Mongo.Database)
		heartBeatKickTask := task.NewHeartBeatTask(utils.DefaultConf)
		recordTaskManager := task.NewRecordTask(utils.DefaultConf)
		repairTask, _ := task.NewRepairTask(utils.DefaultConf)
		baseRoomTask, _ := task.NewBaseRoomTaskService(utils.DefaultConf)
		_ = gocron.Every(1).Hours().Do(interviewTask.TaskForModifyInterviewStatus)
		_ = gocron.Every(1).Minutes().Do(baseRoomTask.StartIdleRoomTask)
		_ = gocron.Every(3).Seconds().Do(recordTaskManager.Start)
		_ = gocron.Every(3).Seconds().Do(heartBeatKickTask.Start)
		_ = gocron.Every(5).Seconds().Do(repairTask.Start)
		_ = gocron.Every(5).Seconds().Do(baseRoomTask.StartTimeoutUserTask)
		<-gocron.Start()
	}()
	// 启动 gin HTTP server。
	r, err := web.NewRouter(&utils.DefaultConf)
	if err != nil {
		log.Fatalf("failed to create gin HTTP server, error %v", err)
	}

	errch := make(chan error, 1)
	go func() {
		httpServerErr := r.Run(utils.DefaultConf.ListenAddr)
		errch <- httpServerErr
	}()

	qC := make(chan os.Signal, 1)
	signal.Notify(qC, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-qC:
		log.Info(s.String())
	case err = <-errch:
		log.Error("db stopped, error", err.Error())
	}

}
