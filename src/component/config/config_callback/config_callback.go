package atframework_component_config_callback

import "log/slog"

type ConfigCallback interface {
	LoadFile(string) ([]byte, error)
	GetLogger() *slog.Logger
}
