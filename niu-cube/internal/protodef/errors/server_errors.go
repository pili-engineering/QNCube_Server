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

package errors

import "encoding/json"

// ServerError 服务端内部错误与非正常返回结果定义
type ServerError struct {
	Code    int    `json:"code"`
	Summary string `json:"summary"`
}

func (e *ServerError) Error() string {
	buf, _ := json.Marshal(e)
	return string(buf)
}

// 各种服务端内部错误的错误码定义。错误码为5位数字。
const (
	// 1开头表示服务端内部，或数据库访问相关的错误。
	ServerErrorUserNotLoggedin      = 10001
	ServerErrorUserLoggedin         = 10002
	ServerErrorUserNoPermission     = 10003
	ServerErrorUserNotfound         = 10004
	ServerErrorRoomNotFound         = 10005
	ServerErrorRoomNameUsed         = 10006
	ServerErrorTooManyRooms         = 10007
	ServerErrorCanOnlyCreateOneRoom = 10008
	ServerErrorUserBroadcasting     = 10009
	ServerErrorUserWatching         = 10010
	ServerErrorSMSSendTooFrequent   = 10011
	ServerErrorUserJoined           = 10012
	ServerErrorMongoOpFail          = 11000
	// 2开头表示外部服务错误。
	ServerErrorSMSSendFail = 20001
)
