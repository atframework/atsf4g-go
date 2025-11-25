package atframework_component_user_controller_action

import (
	// libatapp "github.com/atframework/libatapp-go"

	"time"

	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	"github.com/atframework/libatapp-go"

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

	t.session.UnbindUser(t.GetRpcContext(), t.user)
	uc.GlobalSessionManager.RemoveSession(t.GetRpcContext(), t.session.GetKey(), int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_RESET_BY_PEER), "closed by client")

	// TODO: 等待当前任务执行完毕
	libatapp.AtappGetModule[*uc.UserManager](uc.GetReflectTypeUserManager(), t.GetAwaitableContext().GetApp()).Remove(t.GetAwaitableContext(), t.user.GetZoneId(), t.user.GetUserId(), t.user, false)
	return nil
}

func RemoveSessionAndMaybeLogoutUser(rd cd.DispatcherImpl, ctx cd.RpcContext, sessionKey *uc.SessionKey) {
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

	logoutTask, startData := cd.CreateNoMessageTaskAction(
		rd, ctx, userImpl.GetActorExecutor(),
		func(rd cd.DispatcherImpl, actor *cd.ActorExecutor, timeout time.Duration) *TaskActionUserLogout {
			ta := TaskActionUserLogout{
				TaskActionNoMessageBase: cd.CreateNoMessageTaskActionBase(rd, actor, timeout),
				user:                    userImpl,
				session:                 session,
			}
			return &ta
		},
	)

	err := libatapp.AtappGetModule[*cd.TaskManager](cd.GetReflectTypeTaskManager(), ctx.GetApp()).StartTaskAction(ctx, logoutTask, &startData)
	if err != nil {
		ctx.LogError("TaskActionUserLogout StartTaskAction failed", "error", err,
			"zone_id", userImpl.GetZoneId(), "user_id", userImpl.GetUserId(), "session_id", sessionKey.SessionId, "session_node_id", sessionKey.NodeId)

		session.UnbindUser(ctx, userImpl)
		uc.GlobalSessionManager.RemoveSession(ctx, sessionKey, int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_RESET_BY_PEER), "closed by client")
	}
}
