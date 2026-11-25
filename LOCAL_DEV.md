# 给其他 AI 的最短启动路径

在 PowerShell 中执行，工作目录为 `D:\PVT_AI`。

如果当前终端的 `go` / `bun` 不在 PATH，使用下面的绝对路径：

```powershell
$Go = 'C:\Program Files\Go\bin\go.exe'
$Bun = 'C:\Users\57558\AppData\Local\Microsoft\WinGet\Packages\Oven-sh.Bun_Microsoft.Winget.Source_8wekyb3d8bbwe\bun-windows-x64\bun.exe'
```

当前服务已构建并运行：

```text
http://127.0.0.1:3000
```

重新构建并重启：

```powershell
cd D:\PVT_AI\web
& $Bun install --frozen-lockfile --registry https://registry.npmmirror.com
& $Bun run build

cd D:\PVT_AI
$env:GOPROXY = 'https://goproxy.cn,direct'
& $Go build -trimpath -o .runtime\new-api-local.exe .

& D:\PVT_AI\.runtime\stop-local.ps1
& D:\PVT_AI\.runtime\start-local.ps1
```

如果只是重启，不需要重新构建：

```powershell
& D:\PVT_AI\.runtime\stop-local.ps1
& D:\PVT_AI\.runtime\start-local.ps1
```

验证：

```powershell
Invoke-WebRequest http://127.0.0.1:3000/api/status -UseBasicParsing
Invoke-WebRequest http://127.0.0.1:3000/api/about -UseBasicParsing
```

当前环境：

```text
Go 1.27.4
Bun 1.4.4
运行模式：Windows 本地二进制 + SQLite
Docker：未启用，原因见文末
```

---
# PVT_AI 本地开发与启动说明

本项目是 `new-api` 的本地开发副本。

- 本地目录：`D:\PVT_AI`
- 当前开发提交：`a9a94c907`
- 当前运行模式：Windows 官方 Go 构建 + SQLite 轻量模式
- 服务地址：`http://127.0.0.1:3000`
- 生产/完整 Docker 栈尚未启用，原因见文末

## 1. 需要安装的环境

### 必需

| 环境 | 推荐版本 | 用途 |
|---|---|---|
| Go | 1.27.x，最低 1.26.10 | 编译后端 |
| Bun | 1.4.x | 安装前端依赖、构建前端 |
| Git | 2.x | 代码管理 |

当前机器已验证：

```text
Go 1.27.4
Bun 1.4.4
Node v24.15.0（当前方案不依赖 Node 构建）
```

安装命令：

```powershell
winget install --id GoLang.Go --exact
winget install --id Oven-sh.Bun --exact
```

安装后重新打开终端，验证：

```powershell
go version
bun --version
git --version
```

### 可选

只有需要完整 Docker 栈时才需要：

- Docker Desktop
- WSL2
- BIOS 开启 Intel Virtualization Technology / VT-x

当前机器检查结果为：

```text
Virtualization Enabled In Firmware: No
```

所以 Docker Desktop 暂时不能作为本地运行方案。

## 2. 首次构建

在项目根目录执行：

```powershell
cd D:\PVT_AI

# 1. 安装前端依赖
cd web
bun install --frozen-lockfile --registry https://registry.npmmirror.com

# 2. 构建前端，生成 web\dist
bun run build

# 3. 编译后端
cd ..
$env:GOPROXY = 'https://goproxy.cn,direct'
go build -trimpath -o .runtime\new-api-local.exe .
```

注意：

- 前端改动后必须重新执行 `bun run build`；
- 重新构建前端后，必须再执行 Go build，因为前端产物会嵌入后端二进制；
- 不要直接使用旧的 `.runtime\new-api-win.exe`，它不包含你的源码修改。

## 3. 启动

已准备好本地启动脚本：

```powershell
& D:\PVT_AI\.runtime\start-local.ps1
```

脚本会：

- 使用 `.runtime\new-api-local.exe`
- 读取 `.runtime\new-api.env`
- 使用 SQLite：`D:\PVT_AI\data\new-api.db`
- 监听 `127.0.0.1:3000`
- 日志写入 `.runtime\stdout-local.log` 和 `.runtime\stderr-local.log`

停止：

```powershell
& D:\PVT_AI\.runtime\stop-local.ps1
```

## 4. 验证服务

```powershell
Invoke-WebRequest http://127.0.0.1:3000/api/status -UseBasicParsing
Invoke-WebRequest http://127.0.0.1:3000/api/about -UseBasicParsing
```

或者直接打开：

```text
http://127.0.0.1:3000
```

当前已验证：

- `/api/status` 返回 HTTP 200
- `/api/about` 返回修改后的“关于我们”内容
- 前端页面可以打开

## 5. 本地配置和敏感文件

这些文件只在本机存在，不应提交：

```text
D:\PVT_AI\.env
D:\PVT_AI\.runtime\
D:\PVT_AI\data\
D:\PVT_AI\logs\
```

`data` 中保存 SQLite 数据库，清理前必须先备份：

```powershell
Copy-Item D:\PVT_AI\data\new-api.db D:\PVT_AI\data\new-api.db.bak
```

## 6. 常见问题

### 端口 3000 被占用

```powershell
Get-NetTCPConnection -LocalPort 3000 -State Listen
```

先运行停止脚本，或停止对应 PID。

### 前端依赖下载卡住

使用国内镜像：

```powershell
bun install --frozen-lockfile --registry https://registry.npmmirror.com
```

### 页面看不到前端修改

按顺序重新执行：

```powershell
cd D:\PVT_AI\web
bun run build
cd ..
$env:GOPROXY = 'https://goproxy.cn,direct'
go build -trimpath -o .runtime\new-api-local.exe .
& D:\PVT_AI\.runtime\stop-local.ps1
& D:\PVT_AI\.runtime\start-local.ps1
```

### Go 编译报标准库包缺失

通常是 Go 安装未完成或工具链损坏。先验证：

```powershell
go version
go env GOROOT
```

确认 `GOROOT` 下的标准库完整，必要时重新安装 Go 后重新构建。

## 7. 完整 Docker 栈

仓库已准备：

```text
D:\PVT_AI\docker-compose.local.yml
```

它包含：

- New API
- PostgreSQL
- Redis
- 健康检查
- 固定 New API 版本
- 本地 `.env` 配置

启用 Docker 前需要：

1. 重启进入 BIOS；
2. 开启 Intel VT-x / Virtualization Technology；
3. 安装 Docker Desktop；
4. 安装并启用 WSL2；
5. 执行：

```powershell
cd D:\PVT_AI
docker compose -f docker-compose.local.yml up -d
```

Docker 版本适合做接近生产形态的联调，但当前机器还没有满足虚拟化条件。

## 8. Git 远程

```text
origin   https://github.com/123perfect321/new-api.git
upstream https://github.com/QuantumNous/new-api.git
```

后续本地修改提交到自己的仓库：

```powershell
git add .
git commit -m "your message"
git push origin main
```

同步官方更新时不要把官方源码直接覆盖到本地开发分支，先拉取：

```powershell
git fetch upstream --tags
```

再根据版本策略评估合并。


## 本地 OIDC/SSO 测试登录

当前本地已启用一个仅用于开发的 OIDC 身份源。

```text
New API: http://127.0.0.1:3000
本地 SSO: http://127.0.0.1:5556
登录账号: test
登录密码: Test1234!
OIDC callback: http://127.0.0.1:3000/oauth/oidc
```

一键启动：

```powershell
& D:\PVT_AI\.runtime\start-local-stack.ps1
```

一键停止：

```powershell
& D:\PVT_AI\.runtime\stop-local-stack.ps1
```

本地 SSO 文件位于 `.runtime\local-sso\`，不会提交到 Git。测试用户会被 New API 记录为 `oidc_id=local-user-001`，登录方式为 `oauth:oidc`。

这只是本地 OIDC 测试身份源，不是企业真实 IdP。切换到企业 SSO 时，只需要在 New API 管理台替换 OIDC 配置：

```text
Client ID
Client Secret
Authorization Endpoint
Token Endpoint
UserInfo Endpoint
Well Known URL
```

真实企业 SSO 的 Client Secret 不要写入 Markdown、Git 或聊天记录，放在服务器环境变量或密钥系统中。

## OIDC 部门分组与额度策略

部门名称不写死在代码中。New API 会优先读取 OIDC 的 `org_name`，并兼容 `organization_name`、`department_name`、`org_id` 等常见字段。

部门初始额度可通过以下环境变量配置：

```text
OIDC_DEFAULT_INITIAL_QUOTA_CREDITS=800
OIDC_DEPARTMENT_QUOTAS_JSON={"org-id-1":800,"org-id-2":1200}
```

详细说明见 [docs/oauth-department-policy.md](docs/oauth-department-policy.md)。
