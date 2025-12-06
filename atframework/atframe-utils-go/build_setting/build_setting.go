package buildsetting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Tool 代表单个工具的配置
type Tool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// BuildSettings 是完整的构建配置
type BuildSettings struct {
	Version   string          `json:"version"`
	Timestamp time.Time       `json:"timestamp"`
	Tools     map[string]Tool `json:"tools"`
}

// Manager 管理 BuildSettings 的读写
type Manager struct {
	settingsFile string // 配置文件路径
}

// NewManagerInDir 在指定目录创建 Manager，使用默认文件名 build-settings.json
// 如果文件不存在，则不创建，调用方需要显式调用 Init() 来初始化
func NewManagerInDir(dir string) (*Manager, error) {
	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	settingsFile := filepath.Join(dir, "build-settings.json")
	return &Manager{
		settingsFile: settingsFile,
	}, nil
}

// 用已经存在的配置文件创建 Manager
func NewManagerLoadExistSettingsFile(filePath string) (*Manager, error) {
	return &Manager{
		settingsFile: filePath,
	}, nil
}

// GetSettingsFile 返回配置文件路径
func (m *Manager) GetSettingsFile() string {
	return m.settingsFile
}

// Read 读取配置文件
func (m *Manager) Read() (*BuildSettings, error) {
	data, err := os.ReadFile(m.settingsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 文件不存在，返回 nil
		}
		return nil, fmt.Errorf("failed to read settings: %w", err)
	}

	var settings BuildSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings: %w", err)
	}
	return &settings, nil
}

// Write 写入配置文件
func (m *Manager) Write(settings *BuildSettings) error {
	settings.Timestamp = time.Now()
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(m.settingsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}
	return nil
}

// GetDefaultSettings 返回默认的构建设置
func GetDefaultSettings() *BuildSettings {
	return &BuildSettings{
		Version:   "1.0",
		Timestamp: time.Now(),
		Tools:     make(map[string]Tool),
	}
}

// Init 初始化配置文件（删除旧文件并生成新的）
func (m *Manager) Init() error {
	// 删除老文件（如果存在）
	if err := os.Remove(m.settingsFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old settings file: %w", err)
	}

	// 生成新的默认配置
	settings := GetDefaultSettings()
	if err := m.Write(settings); err != nil {
		return err
	}

	return nil
}

func (m *Manager) SetDocDir(dir string) error {

	m.settingsFile = filepath.Join(dir, "build-settings.json")

	// 检查文件是否已存在
	if _, err := os.Stat(m.settingsFile); err == nil {
		// 文件已存在，不做任何操作
		return nil
	} else if !os.IsNotExist(err) {
		// 其他错误
		return fmt.Errorf("failed to check settings file: %w", err)
	}

	// 文件不存在，创建新的默认配置
	settings := GetDefaultSettings()
	if err := m.Write(settings); err != nil {
		return err
	}

	return nil
}

// SetTool 设置工具路径
func (m *Manager) SetTool(toolName, version, path string) error {
	if toolName == "" || path == "" {
		return fmt.Errorf("input toolName:%s path:%s tool name and path required ", toolName, path)
	}

	settings, err := m.Read()
	if err != nil {
		return err
	}

	if settings == nil {
		return fmt.Errorf("build-settings.json not found. Call Init() or EnsureInit() first")
	}

	// 验证路径是否存在
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("path does not exist: %s", path)
	}

	// 更新工具信息
	tool := Tool{
		Name:    toolName,
		Version: version,
		Path:    path,
	}
	settings.Tools[toolName] = tool

	if err := m.Write(settings); err != nil {
		return err
	}

	return nil
}

// GetToolPath 获取工具路径
func (m *Manager) GetToolPath(toolName string) (string, error) {
	if toolName == "" {
		return "", fmt.Errorf("tool name required")
	}

	settings, err := m.Read()
	if err != nil {
		return "", err
	}

	if settings == nil {
		return "", fmt.Errorf("build-settings.json not found")
	}

	tool, exists := settings.Tools[toolName]
	if !exists {
		return "", fmt.Errorf("tool '%s' not found", toolName)
	}

	if tool.Path == "" {
		return "", fmt.Errorf("tool '%s' has no path configured", toolName)
	}

	// 验证文件是否存在
	if _, err := os.Stat(tool.Path); err != nil {
		return "", fmt.Errorf("tool '%s' path does not exist: %s", toolName, tool.Path)
	}

	return tool.Path, nil
}

// ResetTool 重置工具配置
func (m *Manager) ResetTool(toolName string) error {
	if toolName == "" {
		return fmt.Errorf("tool name required")
	}

	settings, err := m.Read()
	if err != nil {
		return err
	}

	if settings == nil {
		return fmt.Errorf("build-settings.json not found")
	}

	tool, exists := settings.Tools[toolName]
	if !exists {
		return fmt.Errorf("tool '%s' not found", toolName)
	}

	// 重置工具配置
	tool.Path = ""
	tool.Version = ""
	settings.Tools[toolName] = tool

	if err := m.Write(settings); err != nil {
		return err
	}

	return nil
}

// ListTools 列出所有工具（返回映射）
func (m *Manager) ListTools() (map[string]string, error) {
	settings, err := m.Read()
	if err != nil {
		return nil, err
	}

	if settings == nil {
		return nil, fmt.Errorf("build-settings.json not found")
	}

	ret := make(map[string]string)
	var nameStr []byte
	for name, tool := range settings.Tools {

		nameStr, err = json.Marshal(tool)
		if err != nil {
			return nil, err
		}
		ret[name] = string(nameStr)
	}
	return ret, nil
}

// GetTool 获取工具信息
func (m *Manager) GetTool(toolName string) (*Tool, error) {
	if toolName == "" {
		return nil, fmt.Errorf("tool name required")
	}

	settings, err := m.Read()
	if err != nil {
		return nil, err
	}

	if settings == nil {
		return nil, fmt.Errorf("build-settings.json not found")
	}

	tool, exists := settings.Tools[toolName]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", toolName)
	}

	return &tool, nil
}

// VerifyTool 验证工具是否存在
func (m *Manager) VerifyTool(toolName string) (bool, error) {
	tool, err := m.GetTool(toolName)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(tool.Path)
	return err == nil, nil
}
