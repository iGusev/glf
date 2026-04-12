# GLF - GitLab 模糊查找器

<div align="center">
  <strong><a href="README.md">🇬🇧 English</a></strong> | <strong><a href="README_ru.md">🇷🇺 Русский</a></strong> | <strong><a href="README_cn.md">🇨🇳 中文</a></strong>
</div>

<br>

⚡ 使用本地缓存快速在自托管 GitLab 项目中进行模糊搜索的命令行工具。

<div align="center">
  <img src="demo.gif" alt="GLF Demo" />
</div>

[![CI](https://github.com/igusev/glf/workflows/CI/badge.svg)](https://github.com/igusev/glf/actions/workflows/ci.yml)
[![Security](https://github.com/igusev/glf/workflows/Security/badge.svg)](https://github.com/igusev/glf/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/igusev/glf/branch/main/graph/badge.svg)](https://codecov.io/gh/igusev/glf)
[![Go Report Card](https://goreportcard.com/badge/github.com/igusev/glf)](https://goreportcard.com/report/github.com/igusev/glf)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## ✨ 特性

- ⚡ **闪电般快速的模糊搜索**，支持本地缓存
- 🔍 **多关键词搜索** - 使用空格搜索：`"api storage"` 可查找同时包含这两个词的项目
- 🧠 **智能排序** - 经常选择的项目自动排在前面
- 🔁 **启动时自动同步** - 在您搜索时后台刷新项目
- 🔌 **JSON API 模式** - 机器可读输出，适用于 Raycast、Alfred 和自定义集成
- 🌍 **跨平台** 支持 macOS、Linux 和 Windows

## 🚀 快速开始

### 安装

#### Homebrew (macOS/Linux)

在 macOS 或 Linux 上安装 GLF 最简单的方法：

```bash
# 添加 tap
brew tap igusev/tap

# 安装 GLF
brew install glf

# 更新到最新版本
brew upgrade glf
```

#### MacPorts (macOS)

macOS 用户的替代安装方法：

```bash
# 克隆 ports 仓库
git clone https://github.com/igusev/macports-ports.git
cd macports-ports

# 添加为本地 port 源（需要 sudo）
sudo bash -c "echo 'file://$(pwd)' >> /opt/local/etc/macports/sources.conf"

# 更新并安装
sudo port sync
sudo port install glf

# 更新到最新版本
sudo port selfupdate
sudo port upgrade glf
```

#### Scoop (Windows)

在 Windows 上安装 GLF 最简单的方法：

```powershell
# 添加 bucket
scoop bucket add igusev https://github.com/igusev/scoop-bucket

# 安装 GLF
scoop install igusev/glf

# 更新到最新版本
scoop update glf
```

#### 从源码安装

```bash
# 克隆仓库
git clone https://github.com/igusev/glf.git
cd glf

# 构建并安装
make install
```

#### 二进制发布版

您可以从 [releases 页面](https://github.com/igusev/glf/releases) 下载官方 GLF 二进制文件。

支持平台：**macOS** (Intel & Apple Silicon)、**Linux** (x64、ARM、ARM64 等)、**Windows** (x64)、**FreeBSD**、**OpenBSD**。

### 配置

运行交互式配置向导：

```bash
glf --init
```

程序会提示您输入：
- GitLab 实例 URL（例如 `https://gitlab.example.com`）
- 个人访问令牌（需要 `read_api` 权限）
- API 超时时间（默认：30 秒）

配置将保存到 `~/.config/glf/config.yaml`。

重置并重新配置：

```bash
glf --init --reset
```

#### 手动配置

创建 `~/.config/glf/config.yaml`：

```yaml
gitlab:
  url: "https://gitlab.example.com"
  token: "your-personal-access-token"
  timeout: 30  # 可选，默认为 30 秒

cache:
  dir: "~/.cache/glf"  # 可选
```

#### 环境变量

您也可以使用环境变量：

```bash
export GLF_GITLAB_URL="https://gitlab.example.com"
export GLF_GITLAB_TOKEN="your-token-here"
export GLF_GITLAB_TIMEOUT=30  # 可选
```

### 创建个人访问令牌

1. 进入您的 GitLab 实例
2. 导航到 **用户设置** → **访问令牌**
3. 创建一个具有 `read_api` 权限的新令牌
4. 复制令牌并在 `glf --init` 中使用

### 同步项目

从 GitLab 获取项目并构建本地缓存：

```bash
glf sync
```

### 搜索项目

#### 交互模式（默认）

```bash
# 启动交互式模糊查找器
glf

# 使用初始查询启动
glf backend
```

**导航方式：**
- `↑/↓` - 浏览结果
- `Enter` - 选择项目
- `Ctrl+R` - 手动刷新/从 GitLab 同步项目
- `Ctrl+X` - 从搜索结果中排除/取消排除项目
- `Ctrl+H` - 切换显示被排除的项目
- `?` - 切换帮助文本
- `Esc`/`Ctrl+C` - 退出
- 输入以实时过滤项目

**活动指示器：**
- `○` - 空闲（无操作）
- `●`（绿色）- 活动中：正在同步项目或加载选择历史
- `●`（红色）- 错误：同步失败
- 启动时自动同步，可通过 `Ctrl+R` 手动同步

## 📖 使用方法

### 命令

```
glf [query]           搜索项目（默认：交互式 TUI）
glf --init            配置 GitLab 连接
glf --init --reset    重置并重新配置 GitLab 连接
glf --sync            将项目从 GitLab 同步到本地缓存
glf --help            显示帮助
```

### 标志

```
--init                运行交互式配置向导
--reset               重置配置并从头开始（与 --init 一起使用）
-g, --open            --go 的别名（用于兼容性）
--go                  自动选择第一个结果并在浏览器中打开
-s, --sync            同步项目缓存
--full                强制完全同步（与 --sync 一起使用）
-v, --verbose         启用详细日志记录
--scores              显示分数明细以调试排名
--json                以 JSON 格式输出结果（用于 API 集成）
--limit N             限制 JSON 模式下的结果数量（默认：20）
```

### 示例

```bash
# 交互式搜索
glf

# 使用预填充的查询进行搜索
glf microservice

# 多关键词搜索（匹配包含所有词的项目）
glf api storage        # 查找同时包含 "api" 和 "storage" 的项目
glf user auth service  # 查找包含所有三个词的项目

# 自动选择第一个结果并在浏览器中打开
glf ingress -g         # 打开第一个 "ingress" 匹配项
glf api --go           # 与 -g 相同（兼容性别名）

# 在浏览器中打开当前 Git 仓库
glf .

# 从 GitLab 同步项目
glf --sync             # 增量同步
glf --sync --full      # 完全同步（删除已删除的项目）

# 用于调试的详细模式
glf sync --verbose

# 显示用于调试的排名分数
glf --scores

# 配置 GitLab 连接
glf --init

# 重置并重新配置
glf --init --reset
```

### JSON 输出模式（API 集成）

GLF 支持 JSON 输出，可与 Raycast、Alfred 或自定义脚本等工具集成：

```bash
# 以 JSON 格式输出搜索结果
glf --json api

# 限制结果数量
glf --json --limit 5 backend

# 包含相关性分数（可选）
glf --json --scores microservice

# 获取所有项目（无查询）
glf --json --limit 100
```

**JSON 输出格式（不带 --scores）：**

```json
{
  "query": "api",
  "results": [
    {
      "path": "backend/api-server",
      "name": "API Server",
      "description": "REST API for authentication",
      "url": "https://gitlab.example.com/backend/api-server"
    }
  ],
  "total": 5,
  "limit": 20
}
```

**JSON 输出格式（带 --scores）：**

```json
{
  "query": "api",
  "results": [
    {
      "path": "backend/api-server",
      "name": "API Server",
      "description": "REST API for authentication",
      "url": "https://gitlab.example.com/backend/api-server",
      "score": 123.45
    }
  ],
  "total": 5,
  "limit": 20
}
```

**分数明细：**

使用 `--scores` 标志时，每个项目都包含一个相关性分数，该分数综合了以下因素：
- **搜索相关性**：模糊匹配 + 全文搜索分数
- **使用历史**：之前选择的频率（带指数衰减）
- **查询特定提升**：对于使用此确切查询选择的项目，分数乘以 3 倍

分数越高表示匹配度越好。项目自动按分数降序排列。

**用例：**
- **Raycast 扩展**：从 Raycast 快速导航项目
- **Alfred 工作流**：在 Alfred 中搜索 GitLab 项目
- **CI/CD 脚本**：自动化项目发现和 URL 生成
- **自定义工具**：在 GLF 搜索的基础上构建您自己的集成
- **分析**：使用 `--scores` 了解排名并优化搜索查询

**错误处理：**

当发生错误时，GLF 输出 JSON 错误格式并以代码 1 退出：

```json
{
  "error": "no projects in cache"
}
```

### 智能排序

GLF 学习您的选择模式并自动提升经常使用的项目：

- **第一次**：搜索 `"api"` → 选择 `myorg/api/storage`
- **下次**：搜索 `"api"` → `myorg/api/storage` 出现在**第一位**！
- 您选择项目的次数越多，它的排名就越高
- 查询特定提升：为特定搜索词选择的项目对这些词的排名更高
- 最近的选择获得额外提升（最近 7 天）

历史记录存储在 `~/.cache/glf/history.gob` 中，并在会话之间保持。

## 🔧 开发

### 构建

```bash
# 为当前平台构建
make build

# 为所有平台构建
make build-all

# 为特定平台构建
make build-linux
make build-macos
make build-windows

# 创建发布归档
make release
```

### 测试

```bash
# 运行测试
make test

# 运行带覆盖率的测试
make test-coverage

# 格式化代码
make fmt

# 运行代码检查器
make lint
```

### 发布

GLF 通过 GitHub Actions 和 [GoReleaser](https://goreleaser.com/) 使用自动化 CI/CD 进行发布。

#### 自动发布流程

当推送新版本标签时，发布工作流会自动：

1. ✅ 为所有支持的平台构建二进制文件（macOS、Linux、Windows、FreeBSD、OpenBSD）
2. ✅ 创建包含产物和变更日志的 GitHub Release
3. ✅ 更新 [Homebrew tap](https://github.com/igusev/homebrew-tap)，供 macOS/Linux 用户使用
4. ✅ 更新 [MacPorts Portfile](https://github.com/igusev/macports-ports)，供 macOS 用户使用
5. ✅ 更新 [Scoop bucket](https://github.com/igusev/scoop-bucket)，供 Windows 用户使用

#### 创建新版本

```bash
# 创建并推送版本标签
git tag v0.3.0
git push origin v0.3.0

# GitHub Actions 将自动：
# - 运行 GoReleaser
# - 构建跨平台二进制文件
# - 创建 GitHub 发布
# - 更新包管理器（Homebrew、MacPorts、Scoop）
```

#### 手动发布（可选）

您也可以从 GitHub Actions UI 手动触发发布：
- 转到 **Actions** → **Release** → **Run workflow**

### 项目结构

```
glf/
├── cmd/glf/              # CLI 入口点
│   └── main.go           # 主命令和搜索逻辑
├── internal/
│   ├── config/           # 配置处理
│   ├── gitlab/           # GitLab API 客户端
│   ├── history/          # 选择频率跟踪
│   ├── index/            # 描述索引（Bleve）
│   ├── logger/           # 日志工具
│   ├── search/           # 组合模糊 + 全文搜索
│   ├── sync/             # 同步逻辑
│   ├── tui/              # 终端 UI（Bubbletea）
│   └── types/            # 共享类型
├── Makefile              # 构建自动化
└── README.md
```

## ⚙️ 配置选项

### GitLab 设置

| 选项 | 描述 | 默认值 | 必需 |
|------|------|--------|------|
| `gitlab.url` | GitLab 实例 URL | - | 是 |
| `gitlab.token` | 个人访问令牌 | - | 是 |
| `gitlab.timeout` | API 超时时间（秒） | 30 | 否 |

### 缓存设置

| 选项 | 描述 | 默认值 | 必需 |
|------|------|--------|------|
| `cache.dir` | 缓存目录路径 | `~/.cache/glf` | 否 |

### 排除项

| 选项 | 描述 | 默认值 | 必需 |
|------|------|--------|------|
| `exclusions` | 要排除的项目路径列表 | `[]` | 否 |

带排除项的示例：

```yaml
gitlab:
  url: "https://gitlab.example.com"
  token: "your-token"

exclusions:
  - "archived/old-project"
  - "deprecated/legacy-api"
```

可以在 TUI 中使用 `Ctrl+X` 切换排除的项目，或使用 `Ctrl+H` 隐藏/显示它们。

## 🐛 故障排除

### 连接问题

```bash
# 使用详细模式查看详细日志
glf sync --verbose
```

**常见问题：**
- 无效的 GitLab URL：验证配置中的 URL
- 令牌过期：在 GitLab 中重新生成令牌
- 网络超时：增加配置中的超时时间
- 权限不足：确保令牌具有 `read_api` 权限

### 缓存问题

```bash
# 检查缓存位置
ls -la ~/.cache/glf/

# 清除缓存并重新同步
rm -rf ~/.cache/glf/
glf sync
```

### 配置问题

```bash
# 重新配置 GitLab 连接
glf --init

# 重置并从头开始重新配置
glf --init --reset

# 检查当前配置
cat ~/.config/glf/config.yaml
```

## 📝 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 🤝 贡献

欢迎贡献！请随时提交问题和拉取请求。

## 🙏 致谢

- 使用 [Cobra](https://github.com/spf13/cobra) 作为 CLI 框架
- 使用 [Bubbletea](https://github.com/charmbracelet/bubbletea) 提供 UI 支持
- 使用 [Lipgloss](https://github.com/charmbracelet/lipgloss) 进行样式设计
- 使用 [Bleve](https://github.com/blevesearch/bleve) 进行搜索索引
- 通过 [go-gitlab](https://gitlab.com/gitlab-org/api/client-go) 访问 GitLab API
