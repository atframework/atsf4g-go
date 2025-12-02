package atsf4g_go_robot_user

import (
	base "github.com/atframework/atsf4g-go/robot/base"
)

type TaskActionUser struct {
	base.TaskActionBase
	User User
	Fn   func(*TaskActionUser)
}

func init() {
	var _ base.TaskActionImpl = &TaskActionUser{}
}

func (t *TaskActionUser) BeforeYield() {
	t.User.ReleaseActionGuard()
}

func (t *TaskActionUser) AfterYield() {
	t.User.TakeActionGuard()
}

func (t *TaskActionUser) HookRun() {
	t.User.TakeActionGuard()
	defer t.User.ReleaseActionGuard()
	t.Fn(t)
}
