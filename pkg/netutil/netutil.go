package netutil

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type (
	// PortInfo 端口信息
	PortInfo struct {
		Port    int
		PID     int
		Process string
		State   string
	}
)

// IsPortInUse 检查端口是否被占用
func IsPortInUse(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	listener.Close()
	return false
}

// GetPortInfo 获取端口占用信息（macOS/Linux）
func GetPortInfo(port int) (*PortInfo, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-sTCP:LISTEN")
	case "linux":
		cmd = exec.Command("ss", "-tlnp", fmt.Sprintf("sport = :%d", port))
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("端口 %d 未被占用", port)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("端口 %d 未被占用", port)
	}

	// 解析输出
	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return nil, fmt.Errorf("解析端口信息失败")
	}

	info := &PortInfo{
		Port:    port,
		Process: fields[0],
		State:   "LISTEN",
	}

	// 提取 PID
	if runtime.GOOS == "darwin" {
		if len(fields) >= 2 {
			pid, _ := strconv.Atoi(fields[1])
			info.PID = pid
		}
	} else if runtime.GOOS == "linux" {
		// Linux ss 输出格式不同，需要特殊处理
		for _, field := range fields {
			if strings.Contains(field, "pid=") {
				parts := strings.Split(field, "=")
				if len(parts) == 2 {
					pidStr := strings.TrimRight(parts[1], ",")
					pid, _ := strconv.Atoi(pidStr)
					info.PID = pid
				}
			}
		}
	}

	return info, nil
}

// KillPort 杀死占用端口的进程
func KillPort(port int) error {
	info, err := GetPortInfo(port)
	if err != nil {
		return err
	}

	if info.PID == 0 {
		return fmt.Errorf("无法获取进程 PID")
	}

	cmd := exec.Command("kill", "-9", strconv.Itoa(info.PID))
	return cmd.Run()
}

// ListenPorts 列出所有监听端口
func ListenPorts() ([]PortInfo, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-P", "-n")
	case "linux":
		cmd = exec.Command("ss", "-tlnp")
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行命令失败: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	ports := make([]PortInfo, 0)

	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		var port int
		var pid int
		var process string

		if runtime.GOOS == "darwin" {
			// lsof 格式: COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME
			if len(fields) >= 9 {
				process = fields[0]
				pid, _ = strconv.Atoi(fields[1])
				// 从 NAME 字段提取端口（格式: *:8080 或 127.0.0.1:8080）
				name := fields[8]
				parts := strings.Split(name, ":")
				if len(parts) == 2 {
					port, _ = strconv.Atoi(parts[1])
				}
			}
		} else if runtime.GOOS == "linux" {
			// ss 格式不同，需要特殊处理
			continue
		}

		if port > 0 {
			ports = append(ports, PortInfo{
				Port:    port,
				PID:     pid,
				Process: process,
				State:   "LISTEN",
			})
		}
	}

	return ports, nil
}

// TestConnection 测试 TCP 连接
func TestConnection(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	conn.Close()
	return nil
}

// GetLocalIP 获取本机 IP
func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("未找到本机 IP")
}
