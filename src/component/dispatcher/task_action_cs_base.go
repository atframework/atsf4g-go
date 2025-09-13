package atframework_component_dispatcher

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"
)

type TaskActionCSBase[RequestType protoreflect.Message, ResponseType protoreflect.Message] struct {
	TaskActionBase

	request  RequestType
	response ResponseType
}

func (t *TaskActionCSBase[RequestType, ResponseType]) HookRun(_ TaskActionImpl, startData DispatcherStartData) error {
	if csMsg, ok := startData.Message.Instance.(*public_protocol_extension.CSMsg); ok {
		csMsg.BodyBin = []byte{} // Set request data to CSMsg
		_ = t.request            // Use request parameter
	}
	return nil
}
