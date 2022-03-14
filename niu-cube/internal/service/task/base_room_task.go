package task

import (
	"github.com/solutions/niu-cube/internal/service/db"
	"gopkg.in/mgo.v2"
	"time"

	"github.com/qiniu/x/xlog"

	"github.com/solutions/niu-cube/internal/common/utils"
	"github.com/solutions/niu-cube/internal/protodef/model"
	"github.com/solutions/niu-cube/internal/service/dao"
)

type BaseRoomTask struct {
	baseRoom     dao.BaseRoomDaoInterface
	baseUserMic  dao.BaseUserMicDaoInterface
	baseRoomMic  dao.BaseRoomMicDaoInterface
	baseRoomUser dao.BaseRoomUserDaoInterface
	appConfig    db.AppConfigInterface
	xl           *xlog.Logger
}

func NewBaseRoomTaskService(config utils.Config) (*BaseRoomTask, error) {
	baseRoom, err := dao.NewBaseRoomDaoService(nil, config.Mongo)
	if err != nil {
		return nil, err
	}
	baseUserMic, err := dao.NewBaseUserMicDaoService(nil, config.Mongo)
	if err != nil {
		return nil, err
	}
	baseRoomMic, err := dao.NewBaseRoomMicDaoService(nil, config.Mongo)
	if err != nil {
		return nil, err
	}
	baseRoomUser, err := dao.NewBaseRoomUserDaoService(nil, config.Mongo)
	if err != nil {
		return nil, err
	}
	appConfig, err := db.NewAppConfigService(config.IM, nil)
	if err != nil {
		return nil, err
	}
	xl := xlog.New("base-room-task")
	return &BaseRoomTask{
		baseRoom,
		baseUserMic,
		baseRoomMic,
		baseRoomUser,
		appConfig,
		xl,
	}, nil
}

func (t *BaseRoomTask) StartTimeoutUserTask() {
	oldTime := time.Now().UnixMilli() - 10*time.Minute.Milliseconds()
	threshold := time.UnixMilli(oldTime)
	list, err := t.baseRoomUser.ListByHeartbeatTimeout(t.xl, threshold)
	if err != nil {
		t.xl.Error("list base_room_user failed!")
		return
	}
	for _, val := range list {
		t.outline(&val)
	}
}

func (t *BaseRoomTask) StartIdleRoomTask() {
	oldTime := time.Now().UnixMilli() - 1*time.Hour.Milliseconds()
	threshold := time.UnixMilli(oldTime)
	list, err := t.baseRoom.ListByTimeout(t.xl, threshold)
	if err != nil && err != mgo.ErrNotFound {
		t.xl.Error("list base_room for timeout failed!")
		return
	}
	for _, val := range list {
		l, _ := t.baseRoomUser.ListByRoomId(t.xl, val.Id)
		// 如果没人且距离上次修改超过了一个小时，将释放房间
		if len(l) == 0 {
			t.xl.Infof("release room: %s", val.Id)
			val.Status = model.BaseRoomDestroyed
			_ = t.baseRoom.Update(t.xl, &val)
			_ = t.appConfig.DestroyGroupChat(t.xl, val.QiniuIMGroupId)
		}
	}
}

func (t *BaseRoomTask) outline(roomUser *model.BaseRoomUserDo) {
	room, _ := t.baseRoom.Select(nil, roomUser.RoomId)
	if room != nil && room.Creator == roomUser.UserId {
		t.xl.Infof("room creator outline, and the room will be destroyed.")
		room.Status = model.BaseRoomDestroyed
		_ = t.baseRoom.Update(nil, room)
		_ = t.appConfig.DestroyGroupChat(t.xl, room.QiniuIMGroupId)
	}
	userMic, _ := t.baseUserMic.SelectByRoomIdUserId(nil, roomUser.RoomId, roomUser.UserId)
	if userMic != nil {
		userMic.Status = model.BaseUserMicNonHold
		_ = t.baseUserMic.Update(nil, userMic)
		roomMic, _ := t.baseRoomMic.Select(nil, userMic.RoomId, userMic.MicId)
		if roomMic != nil {
			roomMic.Status = model.BaseRoomMicUnused
			_ = t.baseRoomMic.Update(nil, roomMic)
		}
	}
	roomUser.Status = model.BaseRoomUserTimeout
	_ = t.baseRoomUser.Update(nil, roomUser)
}
