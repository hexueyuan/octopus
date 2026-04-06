# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在本仓库中工作时提供指导。

## 项目概述

Octopus 是一个面向个人的 LLM API 聚合与负载均衡服务。它作为代理连接多个 LLM 提供商（OpenAI、Anthropic、Gemini、Volcengine），提供统一的 API 访问、多协议格式转换、多渠道负载均衡以及 Web 管理面板。

## 构建与开发

**环境要求：** Go 1.24.4、Node.js 18+、pnpm

### 后端

```bash
go run main.go start                     # 启动后端服务（监听 :8080）
OCTOPUS_DEBUG=true go run main.go start  # 调试模式
```

### 前端

```bash
cd web && pnpm install && NEXT_PUBLIC_API_BASE_URL="http://127.0.0.1:8080" pnpm run dev
# 前端开发服务 :3000，后端服务 :8080
```

### 生产构建

前端通过静态导出后，经 `static/static.go` 中的 `go:embed` 嵌入到 Go 二进制文件中。

```bash
cd web && pnpm install && pnpm run build && cd ..
mv web/out static/
go build -o octopus ./
```

### Docker 编译部署

`scripts/deploy.sh` 封装了完整的编译和 Docker 部署流程（自动配置 Go 环境、构建前端、交叉编译后端、构建镜像、备份旧容器并启动新容器）。配置项（镜像名、端口、数据目录等）集中在脚本顶部。

```bash
./scripts/deploy.sh all        # 一键完整流程: 构建前端 + 编译后端 + 构建镜像 + 部署
./scripts/deploy.sh build      # 仅构建不部署
./scripts/deploy.sh deploy     # 用已有镜像更新容器
./scripts/deploy.sh rollback   # 回滚到备份容器
./scripts/deploy.sh cleanup    # 确认稳定后清理备份容器
```

部署时旧容器会自动备份为 `octopus-backup`，新容器启动失败会自动回滚。

### 本地启动测试服务

本机 Go 环境存在 `GOROOT` 被设为旧版 `/Users/hexueyuan/Workroot/bin/go1.23.5` 的问题，直接使用 `go build` 会报 `package crypto/sha3 is not in std` 错误。必须显式指定 `GOROOT` 使用 Homebrew 安装的 Go：

```bash
# 1. 编译后端（必须指定 GOROOT 和正确架构）
GOROOT=/opt/homebrew/opt/go/libexec GOARCH=arm64 GOOS=darwin \
  /opt/homebrew/bin/go build -o /tmp/octopus-test ./

# 2. 启动后端（端口由 data/config.json 中 server.port 决定，当前为 8087）
nohup /tmp/octopus-test start > /tmp/octopus-backend.log 2>&1 &

# 3. 启动前端（API 地址指向后端端口）
cd web && NEXT_PUBLIC_API_BASE_URL="http://127.0.0.1:8087" pnpm run dev
# 前端 :3000，后端 :8087
```

**注意事项：**
- 编译前先 `kill` 占用 8087/3000 端口的旧进程：`lsof -i :8087 -t | xargs kill; lsof -i :3000 -t | xargs kill`
- 不要使用 `$GOROOT` 环境变量中的 go（指向 go1.23.5），始终用 `/opt/homebrew/bin/go` 并配合 `GOROOT=/opt/homebrew/opt/go/libexec`
- 不要交叉编译为 linux/amd64，本机为 Apple Silicon (arm64)，必须 `GOARCH=arm64 GOOS=darwin`
- 后端配置文件为 `data/config.json`，端口等参数在此修改，不通过环境变量控制端口

### 测试与代码检查

```bash
# Go 命令同样需要指定 GOROOT
GOROOT=/opt/homebrew/opt/go/libexec /opt/homebrew/bin/go test ./...   # Go 测试
GOROOT=/opt/homebrew/opt/go/libexec /opt/homebrew/bin/go vet ./...    # Go 静态分析
cd web && pnpm run lint                                                # 前端 ESLint 检查
```

## 架构

### 后端（Go + Gin）

入口：`main.go` → `cmd/start.go` 依次启动：加载配置 → 初始化数据库 → 初始化缓存 → 启动 HTTP 服务 → 启动后台任务。

**`internal/` 下的核心包：**

- **relay/** — 核心代理逻辑。接收 API 请求，通过负载均衡器选择渠道，转发至提供商，回传响应流。包含熔断器和跨渠道重试机制。
- **transformer/** — 协议转换层，包含三个子层：
  - `inbound/` — 解析入站请求（OpenAI Chat、OpenAI Responses、Anthropic、Embedding）为 `model.InternalLLMRequest`
  - `outbound/` — 将内部请求转换为目标提供商格式（OpenAI、Anthropic、Gemini、Volcengine）
  - `model/` — 统一的内部 LLM 请求/响应类型
- **relay/balancer/** — 负载均衡（轮询、随机、故障转移、加权），熔断器，会话粘性
- **server/handlers/** — 路由处理器。每个文件通过 `init()` 函数使用 `router.NewGroupRouter()` 自动注册路由
- **server/middleware/** — Gin 中间件（JWT 认证、API Key 认证、CORS、静态文件服务）
- **op/** — 数据操作层（CRUD、缓存、统计聚合）
- **model/** — GORM 模型（Channel、Group、APIKey、User、Stats*、RelayLog 等）
- **db/migrate/** — 编号迁移文件（001.go、002.go、...）
- **task/** — 后台定时任务（价格同步、统计数据刷盘、模型同步）
- **conf/** — 基于 Viper 的配置加载，配置文件 `data/config.json`，环境变量前缀 `OCTOPUS_`

**LLM 代理接口**（API Key 认证）：
- `POST /v1/chat/completions` — OpenAI Chat 格式
- `POST /v1/responses` — OpenAI Responses 格式
- `POST /v1/messages` — Anthropic 格式
- `POST /v1/embeddings` — Embedding 接口
- `GET /v1/models` — 模型列表

**管理 API**（JWT 认证）：`/api/v1/{channel,group,model,apikey,stats,log,setting,user}`

### 前端（Next.js 16 + TypeScript）

位于 `web/` 目录，使用静态导出（next.config.ts 中 `output: "export"`）。

- **UI 组件：** shadcn/ui（Radix + Tailwind CSS 4），Lucide 图标
- **状态管理：** Zustand，stores 位于 `web/src/stores/`
- **数据请求：** TanStack React Query
- **国际化：** next-intl，语言文件位于 `web/public/locale/`（zh_hans、zh_hant、en）
- **API 客户端：** `web/src/api/`，按接口模块划分
- **路由：** 客户端路由位于 `web/src/route/`，支持懒加载
- **React Compiler** 通过 babel 插件启用

### LLM 请求数据流

1. 请求到达代理接口（如 `/v1/chat/completions`）
2. `middleware.APIKeyAuth()` 验证 API Key
3. Inbound 适配器将请求解析为 `InternalLLMRequest`
4. 负载均衡器从匹配的分组中选择渠道（含熔断器/会话粘性）
5. Outbound 适配器将请求转换为目标提供商格式
6. 响应以 SSE 流式或 JSON 非流式方式回传
7. 统计数据在内存中累积，定期批量刷盘至数据库

## 贡献规范

- 每个 PR 只允许包含一个变更主题（一个新功能或一个 Bug 修复）
- AI 辅助生成的代码必须经过人工审查后方可提交
