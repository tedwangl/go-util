# go-util

一个Go语言工具集合项目。

## 项目结构

```
go-util/
├── cmd/              # 主应用程序入口
│   ├── server/       # 服务器应用
│   └── cli/          # 命令行工具
├── internal/         # 私有应用程序和库代码（外部无法导入）
├── pkg/              # 可被外部应用导入的库代码
├── go.mod            # Go模块定义
└── README.md         # 项目说明
```

**目录说明：**
- `cmd/` - 存放应用程序的main包，每个子目录会产生一个可执行文件
- `internal/` - 存放内部使用的代码，外部项目无法导入此目录中的包
- `pkg/` - 存放可被外部项目导入的公共库代码

## 功能特性

- `reverse <text>` - 反转输入文本
- `upper <text>` - 将文本转换为大写
- `lower <text>` - 将文本转换为小写
- `word-count <text>` - 统计文本中的单词数量

## 使用方法

```bash
# 构建服务器应用
cd cmd/server
go build

# 查看帮助
./server help

# 使用示例
./server reverse "hello world"    # 输出: dlrow olleh
./server upper "hello world"      # 输出: HELLO WORLD
./server lower "hello world"      # 输出: hello world
./server word-count "hello world" # 输出: Word count: 2
```

## 如何作为依赖使用

```bash
go get github.com/yourusername/go-util
```

然后在代码中导入：

```go
import "go-util/pkg/utils"

result := utils.Reverse("hello")
```

## 开发

1. 在`internal/`目录下添加仅项目内部使用的包
2. 在`pkg/`目录下添加可导出的公共包（外部可导入）
3. 在`cmd/`目录下添加不同的命令行程序

