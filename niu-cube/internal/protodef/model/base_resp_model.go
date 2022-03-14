package model

type RoomInformation struct {
	BaseRoomDo
	TotalUsers int `json:"totalUsers"`
}

type ListRooms struct {
	List           []RoomInformation `json:"list"`
	Total          int               `json:"total"`
	NextId         string            `json:"nextId"`
	Cnt            int               `json:"cnt"`
	CurrentPageNum int               `json:"currentPageNum"`
	NextPageNum    int               `json:"nextPageNum"`
	PageSize       int               `json:"pageSize"`
	EndPage        bool              `json:"endPage"`
}

type MicInfo struct {
	Uid           string        `json:"uid"`
	UserExtension string        `json:"userExtension"`
	Attrs         []BaseEntryDo `json:"attrs"`
	Params        []BaseEntryDo `json:"params"`
}

type RoomInfoAll struct {
	UserInfo    *BaseUserDo      `json:"userInfo"`
	RoomInfo    *RoomInformation `json:"roomInfo"`
	RtcInfo     *RtcInfoResponse `json:"rtcInfo"`
	Mics        []MicInfo        `json:"mics"`
	AllUserList []BaseUserDo     `json:"allUserList"`
}
