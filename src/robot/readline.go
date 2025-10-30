package main

import (
	"fmt"
	"log"
	"os"
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
	args = strings.Fields(path)
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

func clearHistoryFunc(args []string) string {
	file, _ := os.OpenFile("./cmd_history.tmp", os.O_WRONLY|os.O_TRUNC, 0666)
	file.Close()
	rlIn.ResetHistory()
	fmt.Println(args)
	return ""
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
	RegisterCommand([]string{"clear-history"}, clearHistoryFunc, "")

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

	fmt.Println("Enter 'quit' to Exit")
	fmt.Println(CurrentUser.CmdHelpInfo())

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
