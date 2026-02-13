package conda

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type (
	// Environment conda 环境信息
	Environment struct {
		Name   string `json:"name"`
		Path   string `json:"path"`
		Active bool   `json:"active"`
	}

	// Package 包信息
	Package struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Channel string `json:"channel"`
	}
)

// ListEnvs 列出所有 conda 环境
func ListEnvs() ([]Environment, error) {
	cmd := exec.Command("conda", "env", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行 conda env list 失败: %w", err)
	}

	var result struct {
		Envs []string `json:"envs"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("解析 conda 输出失败: %w", err)
	}

	// 获取当前激活的环境
	activeEnv := os.Getenv("CONDA_DEFAULT_ENV")

	envs := make([]Environment, 0, len(result.Envs))
	for _, path := range result.Envs {
		name := filepath.Base(path)
		if name == "envs" {
			// 跳过 envs 目录本身
			continue
		}
		envs = append(envs, Environment{
			Name:   name,
			Path:   path,
			Active: name == activeEnv,
		})
	}

	return envs, nil
}

// GetCurrentEnv 获取当前激活的环境
func GetCurrentEnv() string {
	env := os.Getenv("CONDA_DEFAULT_ENV")
	if env == "" {
		return "base"
	}
	return env
}

// ListPackages 列出指定环境的包
func ListPackages(envName string) ([]Package, error) {
	cmd := exec.Command("conda", "list", "-n", envName, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行 conda list 失败: %w", err)
	}

	var packages []Package
	if err := json.Unmarshal(output, &packages); err != nil {
		return nil, fmt.Errorf("解析包列表失败: %w", err)
	}

	return packages, nil
}

// InstallPackage 安装包
func InstallPackage(envName, packageName string) error {
	cmd := exec.Command("conda", "install", "-n", envName, "-y", packageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// UninstallPackage 卸载包
func UninstallPackage(envName, packageName string) error {
	cmd := exec.Command("conda", "remove", "-n", envName, "-y", packageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CreateEnv 创建新环境
func CreateEnv(envName, pythonVersion string) error {
	args := []string{"create", "-n", envName, "-y"}
	if pythonVersion != "" {
		args = append(args, fmt.Sprintf("python=%s", pythonVersion))
	}
	cmd := exec.Command("conda", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RemoveEnv 删除环境
func RemoveEnv(envName string) error {
	cmd := exec.Command("conda", "env", "remove", "-n", envName, "-y")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GetPythonPath 获取指定环境的 python 路径
func GetPythonPath(envName string) (string, error) {
	envs, err := ListEnvs()
	if err != nil {
		return "", err
	}

	for _, env := range envs {
		if env.Name == envName {
			pythonPath := filepath.Join(env.Path, "bin", "python")
			if _, err := os.Stat(pythonPath); err == nil {
				return pythonPath, nil
			}
		}
	}

	return "", fmt.Errorf("环境 %s 不存在", envName)
}

// RunPython 在指定环境运行 Python 脚本
func RunPython(envName, script string, args ...string) error {
	pythonPath, err := GetPythonPath(envName)
	if err != nil {
		return err
	}

	cmdArgs := append([]string{script}, args...)
	cmd := exec.Command(pythonPath, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// RunPythonCommand 在指定环境运行 Python 命令
func RunPythonCommand(envName, command string) (string, error) {
	pythonPath, err := GetPythonPath(envName)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(pythonPath, "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("执行失败: %w\n%s", err, out.String())
	}

	return strings.TrimSpace(out.String()), nil
}

// PipInstall 使用 pip 安装包
func PipInstall(envName, packageName string) error {
	pythonPath, err := GetPythonPath(envName)
	if err != nil {
		return err
	}

	pipPath := filepath.Join(filepath.Dir(pythonPath), "pip")
	cmd := exec.Command(pipPath, "install", packageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PipUninstall 使用 pip 卸载包
func PipUninstall(envName, packageName string) error {
	pythonPath, err := GetPythonPath(envName)
	if err != nil {
		return err
	}

	pipPath := filepath.Join(filepath.Dir(pythonPath), "pip")
	cmd := exec.Command(pipPath, "uninstall", "-y", packageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PipList 列出 pip 安装的包
func PipList(envName string) ([]Package, error) {
	pythonPath, err := GetPythonPath(envName)
	if err != nil {
		return nil, err
	}

	pipPath := filepath.Join(filepath.Dir(pythonPath), "pip")
	cmd := exec.Command(pipPath, "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行 pip list 失败: %w", err)
	}

	var packages []Package
	if err := json.Unmarshal(output, &packages); err != nil {
		return nil, fmt.Errorf("解析包列表失败: %w", err)
	}

	return packages, nil
}
