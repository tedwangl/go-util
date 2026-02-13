package cobrax

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// AddCommand 为Command添加子命令
func (c *Command) AddCommand(subcommands ...*Command) {
	for _, subcmd := range subcommands {
		c.Command.AddCommand(subcmd.Command)
	}
}

// AddFlag 添加标志
func (c *Command) AddFlag(name, shorthand string, defaultValue any, usage string) {
	switch val := defaultValue.(type) {
	case string:
		c.Command.Flags().StringP(name, shorthand, val, usage)
	case int:
		c.Command.Flags().IntP(name, shorthand, val, usage)
	case int64:
		c.Command.Flags().Int64P(name, shorthand, val, usage)
	case bool:
		c.Command.Flags().BoolP(name, shorthand, val, usage)
	case []string:
		c.Command.Flags().StringSliceP(name, shorthand, val, usage)
	}
}

// AddFlags 批量添加标志
func (c *Command) AddFlags(flags ...Flag) {
	for _, flag := range flags {
		c.AddFlag(flag.Name, flag.Shorthand, flag.DefaultValue, flag.Usage)
	}
}

// AddPersistentFlag 添加持久化标志（可被子命令继承）
func (c *Command) AddPersistentFlag(name, shorthand string, defaultValue any, usage string) {
	switch val := defaultValue.(type) {
	case string:
		c.Command.PersistentFlags().StringP(name, shorthand, val, usage)
	case int:
		c.Command.PersistentFlags().IntP(name, shorthand, val, usage)
	case int64:
		c.Command.PersistentFlags().Int64P(name, shorthand, val, usage)
	case bool:
		c.Command.PersistentFlags().BoolP(name, shorthand, val, usage)
	case []string:
		c.Command.PersistentFlags().StringSliceP(name, shorthand, val, usage)
	}
}

// AddPersistentFlags 批量添加持久化标志
func (c *Command) AddPersistentFlags(flags ...Flag) {
	for _, flag := range flags {
		c.AddPersistentFlag(flag.Name, flag.Shorthand, flag.DefaultValue, flag.Usage)
	}
}

// SetPersistentPreRunE 设置全局前置钩子（会被子命令继承）
// 常用于：初始化配置、连接数据库、验证权限
func (c *Command) SetPersistentPreRunE(fn func(cmd *cobra.Command, args []string) error) {
	c.Command.PersistentPreRunE = fn
}

// SetPreRunE 设置前置钩子（仅当前命令）
func (c *Command) SetPreRunE(fn func(cmd *cobra.Command, args []string) error) {
	c.Command.PreRunE = fn
}

// SetPersistentPostRunE 设置全局后置钩子（会被子命令继承）
// 常用于：清理资源、关闭连接、记录日志
func (c *Command) SetPersistentPostRunE(fn func(cmd *cobra.Command, args []string) error) {
	c.Command.PersistentPostRunE = fn
}

// ChainPersistentPreRunE 链式设置 PersistentPreRunE，自动调用父命令的钩子
// 使用示例：
//
//	dbCmd.ChainPersistentPreRunE(tool.GetRootCommand(), func(cmd *cobra.Command, args []string) error {
//	    // 这里的代码会在父钩子执行后运行
//	    host := viper.GetString("host")
//	    return nil
//	})
func (c *Command) ChainPersistentPreRunE(parent *Command, fn func(cmd *cobra.Command, args []string) error) {
	parentPreRunE := parent.PersistentPreRunE

	c.Command.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// 先执行父命令的钩子
		if parentPreRunE != nil {
			if err := parentPreRunE(cmd, args); err != nil {
				return err
			}
		}

		// 再执行当前命令的钩子
		if fn != nil {
			return fn(cmd, args)
		}

		return nil
	}
}

// ChainPersistentPostRunE 链式设置 PersistentPostRunE，自动调用父命令的钩子
func (c *Command) ChainPersistentPostRunE(parent *Command, fn func(cmd *cobra.Command, args []string) error) {
	parentPostRunE := parent.PersistentPostRunE

	c.Command.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		// 先执行当前命令的钩子
		if fn != nil {
			if err := fn(cmd, args); err != nil {
				return err
			}
		}

		// 再执行父命令的钩子
		if parentPostRunE != nil {
			return parentPostRunE(cmd, args)
		}

		return nil
	}
}

// SetPostRunE 设置后置钩子（仅当前命令）
func (c *Command) SetPostRunE(fn func(cmd *cobra.Command, args []string) error) {
	c.Command.PostRunE = fn
}

// InheritPersistentFlags 从父命令继承持久化标志
// 注意：Cobra 默认会自动继承，此方法用于显式控制
func (c *Command) InheritPersistentFlags(parent *Command) {
	c.Command.InheritedFlags().AddFlagSet(parent.Command.PersistentFlags())
}

// NewCommandGroup 创建命令组
func NewCommandGroup(name string) *CommandGroup {
	return &CommandGroup{
		Name:     name,
		Commands: []*Command{},
	}
}

// AddCommand 添加命令到组
func (g *CommandGroup) AddCommand(cmds ...*Command) {
	if len(cmds) == 0 {
		return
	}
	g.Commands = append(g.Commands, cmds...)
	for _, cmd := range cmds {
		cmd.GroupID = g.Name
	}
}

// GetFormattedHelp 获取格式化的帮助信息
func GetFormattedHelp(cmd *cobra.Command) string {
	buf := new(strings.Builder)
	cmd.SetOut(buf)
	cmd.HelpFunc()(cmd, []string{})
	return buf.String()
}

// CustomHelpFunc 创建自定义帮助函数
func CustomHelpFunc(customHelp func(cmd *cobra.Command) string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		fmt.Println(customHelp(cmd))
	}
}

// PrintCommandTree 打印命令树形结构
func (t *Tool) PrintCommandTree() string {
	return printTree(t.rootCmd.Command, "", true, true)
}

// printTree 递归打印命令树
func printTree(cmd *cobra.Command, prefix string, isLast bool, isRoot bool) string {
	result := ""

	// 打印当前命令
	if isRoot {
		result += fmt.Sprintf("%s\n", cmd.Name())
	} else {
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		groupInfo := ""
		if cmd.GroupID != "" {
			groupInfo = fmt.Sprintf(" [group:%s]", cmd.GroupID)
		}

		result += fmt.Sprintf("%s%s%s%s\n", prefix, connector, cmd.Name(), groupInfo)
	}

	// 获取子命令
	subCmds := cmd.Commands()
	if len(subCmds) == 0 {
		return result
	}

	// 过滤掉内置命令和 tree 命令
	var filteredCmds []*cobra.Command
	for _, subCmd := range subCmds {
		if subCmd.Name() != "completion" && subCmd.Name() != "help" && subCmd.Name() != "tree" && subCmd.Name() != "version" {
			filteredCmds = append(filteredCmds, subCmd)
		}
	}

	// 按组分类命令
	type groupCommands struct {
		name     string
		commands []*cobra.Command
	}

	groupMap := make(map[string]*groupCommands)
	var ungrouped []*cobra.Command
	var groupOrder []string

	for _, subCmd := range filteredCmds {
		if subCmd.GroupID != "" {
			if _, exists := groupMap[subCmd.GroupID]; !exists {
				groupMap[subCmd.GroupID] = &groupCommands{name: subCmd.GroupID}
				groupOrder = append(groupOrder, subCmd.GroupID)
			}
			groupMap[subCmd.GroupID].commands = append(groupMap[subCmd.GroupID].commands, subCmd)
		} else {
			ungrouped = append(ungrouped, subCmd)
		}
	}

	// 先打印分组命令
	totalGroups := len(groupOrder) + len(ungrouped)
	currentIndex := 0

	for _, groupID := range groupOrder {
		group := groupMap[groupID]
		isLastGroup := currentIndex == totalGroups-1

		connector := "├── "
		if isLastGroup {
			connector = "└── "
		}

		var newPrefix string
		if isRoot {
			newPrefix = ""
		} else if isLast {
			newPrefix = prefix + "    "
		} else {
			newPrefix = prefix + "│   "
		}

		// 打印组名
		result += fmt.Sprintf("%s%s[%s]\n", newPrefix, connector, groupID)

		// 打印组内命令
		groupPrefix := newPrefix
		if isLastGroup {
			groupPrefix += "    "
		} else {
			groupPrefix += "│   "
		}

		for i, subCmd := range group.commands {
			isLastInGroup := i == len(group.commands)-1
			result += printTree(subCmd, groupPrefix, isLastInGroup, false)
		}

		currentIndex++
	}

	// 再打印未分组命令
	for _, subCmd := range ungrouped {
		isLastChild := currentIndex == totalGroups-1

		var newPrefix string
		if isRoot {
			newPrefix = ""
		} else if isLast {
			newPrefix = prefix + "    "
		} else {
			newPrefix = prefix + "│   "
		}

		result += printTree(subCmd, newPrefix, isLastChild, false)
		currentIndex++
	}

	return result
}
