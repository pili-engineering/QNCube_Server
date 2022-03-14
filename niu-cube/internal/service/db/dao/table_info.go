package dao

const (
	// CollectionAccount 存储账号信息的表。
	CollectionAccount = "accounts"
	// CollectionAccountToken 存储已登录用户的表。
	CollectionAccountToken = "account_token"

	// CollectionSMSCode 存储已发送的短信验证码的表。
	CollectionSMSCode = "sms_code"

	// CollectionRoom 存储直播房间信息的表。
	CollectionRoom        = "rooms"
	CollectionRoomAccount = "room_accounts"

	// CollectionBizExtra 每个room关联的业务部分
	CollectionBizExtra = "biz_extras"

	InterviewCollection     = "interviews"
	InterviewUserCollection = "interview_users"

	// CounterCollection 存储各类对象编号的表，用于生成类自增的ID。
	CounterCollection = "_counter"
	TaskCollection    = "task_results"

	// ActionCollection 全局日志流水
	ActionCollection = "actions"

	// CollectionRepairRoom 检修相关业务
	CollectionRepairRoom     = "repair_room"
	CollectionRepairRoomUser = "repair_room_user"

	// CollectionBaseRoom 通用房间
	CollectionBaseRoom     = "base_room"
	CollectionBaseUser     = "base_user"
	CollectionBaseMic      = "base_mic"
	CollectionBaseRoomUser = "base_room_user"
	CollectionBaseRoomMic  = "base_room_mic"
	CollectionBaseUserMic  = "base_user_mic"

	// CollectionSong KTV场景
	CollectionSong         = "song"
	CollectionRoomUserSong = "room_user_song"

	// CollectionMovie 一起看电影相关
	CollectionMovie         = "movie"
	CollectionRoomUserMovie = "room_user_movie"

	// CollectionQiniuIMUser 七牛IM用户信息表
	CollectionQiniuIMUser = "qiniu_im_user"

	CollectionQiniuImageFile = "image_file"

	CollectionExam         = "exam"
	CollectionQuestion     = "exam_question"
	CollectionExamPaper    = "exam_paper"
	CollectionUserExam     = "exam_user"
	CollectionAnswerPaper  = "exam_answer_paper"
	CollectionCheatingExam = "exam_cheating"
	CollectionAppVersion   = "app_version"
)
