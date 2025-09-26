package lobbysvr_data

import (
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
)

type ItemFlowReason struct {
	MajorReason int32
	MinorReason int32
	Parameter   int64
}

type UserItemManagerImpl interface {
	GetOwner() *User

	AddItem(itemOffset *public_protocol_common.DItemOffset, reason *ItemFlowReason) error
	SubItem(itemOffset *public_protocol_common.DItemOffset, reason *ItemFlowReason) error
}

type UserItemManagerBase struct {
	owner *User
}
