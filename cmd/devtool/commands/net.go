package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/tedwangl/go-util/pkg/cobrax"
)

// RegisterNetCommands 注册网络工具相关命令
func RegisterNetCommands(tool *cobrax.Tool) {
	netGroup := cobrax.NewCommandGroup("net")

	// netstat - 网络统计
	netstatCmd := tool.NewCommand(
		"netstat",
		"网络统计工具",
		"netstat 命令快捷方式",
		nil,
	)
	netstatCmd.Command.GroupID = "net"

	// netstat -tuln - 查看监听端口
	netstatListenCmd := tool.NewCommand(
		"listen",
		"查看所有监听端口",
		"netstat -tuln",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			c := exec.Command("netstat", "-tuln")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// netstat -tunap - 查看所有连接
	netstatAllCmd := tool.NewCommand(
		"all",
		"查看所有连接",
		"netstat -tunap",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			c := exec.Command("netstat", "-tunap")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// netstat -r - 查看路由表
	netstatRouteCmd := tool.NewCommand(
		"route",
		"查看路由表",
		"netstat -r",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			c := exec.Command("netstat", "-r")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	netstatCmd.Command.AddCommand(netstatListenCmd.Command, netstatAllCmd.Command, netstatRouteCmd.Command)

	// ss - 查看 socket 统计
	ssCmd := tool.NewCommand(
		"ss",
		"socket 统计工具",
		"ss 命令快捷方式",
		nil,
	)
	ssCmd.Command.GroupID = "net"

	// ss -tuln - 查看所有监听端口
	ssListenCmd := tool.NewCommand(
		"listen",
		"查看所有监听端口",
		"ss -tuln",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			c := exec.Command("ss", "-tuln")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// ss -tunap - 查看所有连接
	ssAllCmd := tool.NewCommand(
		"all",
		"查看所有连接",
		"ss -tunap",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			c := exec.Command("ss", "-tunap")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// ss -s - 统计信息
	ssStatsCmd := tool.NewCommand(
		"stats",
		"显示统计信息",
		"ss -s",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			c := exec.Command("ss", "-s")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	ssCmd.Command.AddCommand(ssListenCmd.Command, ssAllCmd.Command, ssStatsCmd.Command)

	// nc - netcat 工具
	ncCmd := tool.NewCommand(
		"nc",
		"netcat 网络工具",
		"nc 命令快捷方式",
		nil,
	)
	ncCmd.Command.GroupID = "net"

	// nc -zv host port - 测试端口
	ncTestCmd := tool.NewCommand(
		"test",
		"测试端口连通性",
		"nc -zv <主机> <端口>",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("用法: devtool nc test <主机> <端口>")
			}

			host := args[0]
			port := args[1]

			c := exec.Command("nc", "-zv", host, port)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// nc -l port - 监听端口
	ncListenCmd := tool.NewCommand(
		"listen",
		"监听端口",
		"nc -l <端口>",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("用法: devtool nc listen <端口>")
			}

			port := args[0]
			fmt.Printf("监听端口 %s...\n", port)
			c := exec.Command("nc", "-l", port)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Stdin = os.Stdin
			return c.Run()
		}),
	)

	ncCmd.Command.AddCommand(ncTestCmd.Command, ncListenCmd.Command)

	// tcpdump - 抓包工具
	tcpdumpCmd := tool.NewCommand(
		"tcpdump",
		"抓包工具",
		"tcpdump 命令快捷方式",
		nil,
	)
	tcpdumpCmd.Command.GroupID = "net"

	// tcpdump -i any port 80 - 抓取指定端口
	tcpdumpPortCmd := tool.NewCommand(
		"port",
		"抓取指定端口流量",
		"tcpdump -i <网卡> port <端口>",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("用法: devtool tcpdump port <端口> [网卡名，默认 any]")
			}

			port := args[0]
			iface := "any"
			if len(args) > 1 {
				iface = args[1]
			}

			fmt.Printf("抓取网卡 %s 端口 %s 的流量...\n", iface, port)
			c := exec.Command("tcpdump", "-i", iface, "port", port)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// tcpdump -i any host 1.1.1.1 - 抓取指定主机
	tcpdumpHostCmd := tool.NewCommand(
		"host",
		"抓取指定主机流量",
		"tcpdump -i <网卡> host <主机>",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("用法: devtool tcpdump host <主机> [网卡名，默认 any]")
			}

			host := args[0]
			iface := "any"
			if len(args) > 1 {
				iface = args[1]
			}

			fmt.Printf("抓取网卡 %s 主机 %s 的流量...\n", iface, host)
			c := exec.Command("tcpdump", "-i", iface, "host", host)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// tcpdump -i any -w file.pcap - 保存到文件
	tcpdumpSaveCmd := tool.NewCommand(
		"save",
		"保存抓包到文件",
		"tcpdump -i <网卡> -w <文件>",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("用法: devtool tcpdump save <文件名> [网卡名，默认 any]")
			}

			filename := args[0]
			iface := "any"
			if len(args) > 1 {
				iface = args[1]
			}

			fmt.Printf("抓取网卡 %s 的流量并保存到 %s...\n", iface, filename)
			c := exec.Command("tcpdump", "-i", iface, "-w", filename)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	tcpdumpCmd.Command.AddCommand(tcpdumpPortCmd.Command, tcpdumpHostCmd.Command, tcpdumpSaveCmd.Command)

	// nmap - 端口扫描
	nmapCmd := tool.NewCommand(
		"nmap",
		"端口扫描工具",
		"nmap 命令快捷方式",
		nil,
	)
	nmapCmd.Command.GroupID = "net"

	// nmap -sT host - TCP 扫描
	nmapTcpCmd := tool.NewCommand(
		"tcp",
		"TCP 端口扫描",
		"nmap -sT <主机>",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("用法: devtool nmap tcp <主机>")
			}

			host := args[0]
			fmt.Printf("扫描主机 %s 的 TCP 端口...\n", host)
			c := exec.Command("nmap", "-sT", host)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// nmap -sU host - UDP 扫描
	nmapUdpCmd := tool.NewCommand(
		"udp",
		"UDP 端口扫描",
		"nmap -sU <主机>",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("用法: devtool nmap udp <主机>")
			}

			host := args[0]
			fmt.Printf("扫描主机 %s 的 UDP 端口...\n", host)
			c := exec.Command("nmap", "-sU", host)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// nmap -p 80,443 host - 扫描指定端口
	nmapPortCmd := tool.NewCommand(
		"port",
		"扫描指定端口",
		"nmap -p <端口> <主机>",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("用法: devtool nmap port <端口> <主机>")
			}

			port := args[0]
			host := args[1]
			fmt.Printf("扫描主机 %s 的端口 %s...\n", host, port)
			c := exec.Command("nmap", "-p", port, host)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// nmap -sn 192.168.1.0/24 - 主机发现
	nmapPingCmd := tool.NewCommand(
		"ping",
		"主机发现（ping 扫描）",
		"nmap -sn <网段>",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("用法: devtool nmap ping <网段，如 192.168.1.0/24>")
			}

			network := args[0]
			fmt.Printf("扫描网段 %s 的存活主机...\n", network)
			c := exec.Command("nmap", "-sn", network)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	nmapCmd.Command.AddCommand(nmapTcpCmd.Command, nmapUdpCmd.Command, nmapPortCmd.Command, nmapPingCmd.Command)

	netGroup.AddCommand(netstatCmd, ssCmd, ncCmd, tcpdumpCmd, nmapCmd)
	tool.AddGroupLogic(netGroup)
}
