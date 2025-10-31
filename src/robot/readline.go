package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/chzyer/readline"
)

type CommandFunc func([]string) string

type CommandNode struct {
	Children map[string]*CommandNode
	Name     string
	Func     CommandFunc
}

var root = &CommandNode{Children: make(map[string]*CommandNode)}
var rlIn *readline.Instance

func RegisterCommand(path []string, fn CommandFunc, info string) {
	current := root
	for _, key := range path {
		if current.Children[strings.ToLower(key)] == nil {
			current.Children[strings.ToLower(key)] = &CommandNode{Children: make(map[string]*CommandNode)}
			current.Children[strings.ToLower(key)].Name = key
		}
		current = current.Children[strings.ToLower(key)]
	}
	current.Func = fn
	if info != "" {
		if current.Children["-h"] == nil {
			current.Children["-h"] = &CommandNode{
				Children: make(map[string]*CommandNode),
				Name:     "-h",
				Func: func([]string) string {
					return info
				},
			}
			current.Children["-h"].Name = "-h"
		}
	}
}

// FindCommand 根据路径查找命令节点
func FindCommand(path string) (args []string, node *CommandNode) {
	node = root
	args = splitArgs(path)
	for {
		if len(node.Children) == 0 {
			break
		}
		if len(args) == 0 {
			// 没有参数了
			return
		}
		next, ok := node.Children[strings.ToLower(args[0])]
		if !ok {
			return
		}
		node = next
		args = args[1:]
	}
	return
}

// 构建自动补全器
func NewCompleter() *readline.PrefixCompleter {
	return buildCompleterFromNode(root, "")
}

// 递归构建 PrefixCompleter
func buildCompleterFromNode(node *CommandNode, name string) *readline.PrefixCompleter {
	if len(node.Children) == 0 {
		return readline.PcItem(name)
	}
	items := []readline.PrefixCompleterInterface{}

	// 排序一下
	sortKey := make([]string, 0)
	for key, _ := range node.Children {
		sortKey = append(sortKey, key)
	}
	sort.Slice(sortKey, func(i, j int) bool {
		return sortKey[i] < sortKey[j]
	})

	for _, child := range sortKey {
		items = append(items, buildCompleterFromNode(node.Children[child], node.Children[child].Name))
	}
	if name == "" {
		return readline.NewPrefixCompleter(items...)
	}
	return readline.PcItem(name, items...)
}

func splitArgs(input string) []string {
	var result []string
	var current strings.Builder
	var quote rune   // 当前是否在引号内 (' 或 ")
	escaped := false // 上一个字符是否为 '\'

	for _, r := range input {
		switch {
		case escaped:
			// 转义状态下，直接写入字符
			current.WriteRune(r)
			escaped = false

		case r == '\\':
			// 遇到反斜杠，开启转义模式
			escaped = true

		case quote != 0:
			// 在引号内
			if r == quote {
				// 结束引号
				quote = 0
			} else {
				current.WriteRune(r)
			}

		case r == '"' || r == '\'':
			// 开始新的引号块
			quote = r

		case r == ' ' || r == '\t':
			// 空白分隔符（仅在非引号内生效）
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}

		default:
			current.WriteRune(r)
		}
	}

	// 收尾
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// ExecuteCommand 执行命令
func ExecuteCommand(rl *readline.Instance, input string) {
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return
	}
	args, node := FindCommand(input)
	if node == nil {
		fmt.Println("未知命令:", input)
		return
	}

	if node.Func != nil {
		result := node.Func(args)
		if result != "" {
			fmt.Println(result)
		}
	} else {
		fmt.Println("这是一个命令组，不可直接执行")
	}
}

func ReadLine() {
	// 注册命令
	RegisterCommand([]string{"quit"}, nil, "")

	config := &readline.Config{
		Prompt:       "\033[32m»\033[0m ", // 设置提示符
		AutoComplete: NewCompleter(),      // 设置自动补全
		HistoryFile:  "./cmd_history.tmp",
	}

	rlIn, err := readline.NewEx(config)
	if err != nil {
		log.Println("无法创建 readline 实例:", err)
		return
	}
	defer rlIn.Close()

	fmt.Println("Enter 'quit' to Exit, 'Tab' to AutoComplete")

	for {
		cmd, err := rlIn.Readline()
		if err != nil {
			continue
		}
		cmd = strings.TrimSpace(cmd)
		if cmd == "quit" {
			break
		}
		ExecuteCommand(rlIn, cmd)
	}
}
