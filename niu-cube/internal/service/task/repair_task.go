package task

import (
	"github.com/qiniu/x/xlog"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/service/db"
)

type RepairTask struct {
	repair db.RepairInterface
	xl     *xlog.Logger
}

func NewRepairTask(conf utils.Config) (*RepairTask, error) {

	xl := xlog.New("repairTask task")
	repair, err := db.NewRepairService(*conf.Mongo, nil)
	if err != nil {
		panic(err)
	}
	return &RepairTask{
		repair: repair,
		xl:     xl,
	}, nil
}

func (h *RepairTask) Start() {

	// 查看状态是正常的没有心跳的room_user
	users, err := h.repair.ListHeartBeatTimeOutUser(h.xl)
	if err != nil {
		h.xl.Errorf("error list heartbeat timeout user:%v", err)
		return
	}
	for _, user := range users {
		leaveRoomErr := h.repair.LeaveRoom(h.xl, user.UserID, user.RoomId)
		if leaveRoomErr != nil {
			h.xl.Errorf("failed LeaveRoom  userId:%s,roomId:%s, err:%v", user.UserID, user.RoomId, leaveRoomErr)
		} else {
			h.xl.Infof("RepairTask success LeaveRoom  userId:%v,roomId:%v ", user.UserID, user.RoomId)
		}

	}
}
