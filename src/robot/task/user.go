package atsf4g_go_robot_task

import (
	"fmt"
	"time"

	user_data "github.com/atframework/atsf4g-go/robot/data"

	protocol "github.com/atframework/atsf4g-go/robot/protocol"
	lobysvr_protocol_pbdesc "github.com/atframework/atsf4g-go/service-lobbysvr/protocol/public/protocol/pbdesc"
)

func LoginTask(task *user_data.TaskActionUser) (err error) {
	errCode, rsp, rpcErr := protocol.LoginAuthRpc(task)
	if rpcErr != nil {
		err = rpcErr
		return
	}
	if errCode < 0 {
		err = fmt.Errorf("login auth failed, errCode: %d", errCode)
		return
	}

	user := task.User

	if rsp.GetLoginCode() != "" {
		user.SetLoginCode(rsp.GetLoginCode())
	}
	if rsp.GetUserId() != 0 {
		user.SetUserId(rsp.GetUserId())
	}
	if rsp.GetZoneId() != 0 {
		user.SetZoneId(rsp.GetZoneId())
	}

	var loginRsp *lobysvr_protocol_pbdesc.SCLoginRsp
	errCode, loginRsp, rpcErr = protocol.LoginRpc(task)
	if rpcErr != nil {
		task.Log("user login failed, error: %v, open_id: %s, user_id: %d, zone_id: %d", err, user.GetOpenId(), user.GetUserId(), user.GetZoneId())
		err = rpcErr
		return
	}
	if errCode < 0 {
		err = fmt.Errorf("login req failed, errCode: %d", errCode)
		return
	}

	if loginRsp.GetZoneId() != 0 {
		user.SetZoneId(loginRsp.GetZoneId())
	}
	user.SetLogined(true)

	if loginRsp.GetHeartbeatInterval() > 0 {
		user.SetHeartbeatInterval(time.Duration(loginRsp.GetHeartbeatInterval()) * time.Second)
	}

	// 创建Ping流程
	user.CheckPingTask()
	return
}

func LogoutTask(task *user_data.TaskActionUser) (err error) {
	task.User.Logout()
	task.Log("user %s logout", task.User.GetOpenId())
	return nil
}
