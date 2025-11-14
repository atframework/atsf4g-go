package atframework_component_user_controller_action

import (
	// libatapp "github.com/atframework/libatapp-go"
	"context"

	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	uc "github.com/atframework/atsf4g-go/component-user_controller"
)

type TaskActionUserLogout struct {
	cd.TaskActionNoMessageBase

	user    uc.UserImpl
	session *uc.Session
}

func (t *TaskActionUserLogout) Name() string {
	return "TaskActionUserLogout"
}

func (t *TaskActionUserLogout) Run(_startData *cd.DispatcherStartData) error {
	t.LogInfo("TaskActionUserLogout Run", "zone_id", t.user.GetZoneId(), "user_id", t.user.GetUserId(),
		"session_id", t.session.GetKey().SessionId, "session_node_id", t.session.GetKey().NodeId)

	userWritable := t.user.IsWriteable()

	t.session.UnbindUser(t.GetRpcContext(), t.user)
	uc.GlobalSessionManager.RemoveSession(t.GetRpcContext(), t.session.GetKey(), int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_RESET_BY_PEER), "closed by client")

	// TODO: 等待当前任务执行完毕

	if userWritable {
		uc.GlobalUserManager.Remove(t.GetRpcContext(), t.user.GetZoneId(), t.user.GetUserId(), t.user, false)
	}
	return nil
}

func RemoveSessionAndMaybeLogoutUser(rd cd.DispatcherImpl, ctx *cd.RpcContext, sessionKey *uc.SessionKey) {
	session := uc.GlobalSessionManager.GetSession(sessionKey)

	userCSImpl := session.GetUser()
	if userCSImpl == nil {
		session.UnbindUser(ctx, nil)
		uc.GlobalSessionManager.RemoveSession(ctx, sessionKey, int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_RESET_BY_PEER), "closed by client")
		return
	}

	userImpl, ok := userCSImpl.(uc.UserImpl)
	if !ok {
		session.UnbindUser(ctx, nil)
		uc.GlobalSessionManager.RemoveSession(ctx, sessionKey, int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_RESET_BY_PEER), "closed by client")
		return
	}

	if userImpl == nil {
		session.UnbindUser(ctx, nil)
		uc.GlobalSessionManager.RemoveSession(ctx, sessionKey, int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_RESET_BY_PEER), "closed by client")
		return
	}

	logoutTask := &TaskActionUserLogout{
		TaskActionNoMessageBase: cd.CreateTaskActionNoMessageBase(rd, userImpl.GetActorExecutor()),
		user:                    userImpl,
		session:                 session,
	}

	rpcContext := rd.CreateRpcContext()
	if ctx != nil && ctx.Context != nil {
		rpcContext.Context, rpcContext.CancelFn = context.WithCancel(ctx.Context)
	}
	rpcContext.BindAction(logoutTask)

	startData := &cd.DispatcherStartData{
		Message:           nil,
		PrivateData:       nil,
		MessageRpcContext: rpcContext,
	}

	err := cd.RunTaskAction(rd.GetApp(), logoutTask, startData)
	if err != nil {
		rd.GetApp().GetDefaultLogger().Error("TaskActionUserLogout RunTaskAction failed", "error", err,
			"zone_id", userImpl.GetZoneId(), "user_id", userImpl.GetUserId(), "session_id", sessionKey.SessionId, "session_node_id", sessionKey.NodeId)

		session.UnbindUser(ctx, userImpl)
		uc.GlobalSessionManager.RemoveSession(ctx, sessionKey, int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_RESET_BY_PEER), "closed by client")
	}
}
