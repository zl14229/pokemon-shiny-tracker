# Go 后端开发（T001）

后端 Go module 位于仓库根目录，module 路径为 `github.com/zl14229/pokemon-shiny-tracker`。T001 提供可启动的工程骨架，T002 增加正式 Router、`/api/v1` Health API 和统一错误处理；T005 建立 SQLite 连接、模型与版本化迁移，详见 DATABASE.md。

## 工具链与依赖

- Go **1.27.3**：Debian WSL 本机 `/usr/local/go/bin/go` 已安装；本次直接验证为 `linux/amd64`。该版本是 [Go 官方发布记录](https://go.dev/doc/devel/release)中的稳定修订版（2026-10-19 发布），满足 Gin 的 Go 版本要求。`go.mod` 与 Actions 的 `go-version-file` 共用此版本，避免本地/CI 漂移。
- Gin **v1.12.0**、GORM **v1.31.2**：经 Go module 元数据及各自[发布记录](https://github.com/gin-gonic/gin/releases)、[发布记录](https://github.com/go-gorm/gorm/releases)核实，直接依赖及间接依赖版本写入 `go.mod`，校验和写入 `go.sum`。不使用浮动的 `@latest` 构建。
- Make、Python 3.11+、Git：Make 提供可重复命令；Python 标准库用于真实进程启动检查和现有仓库治理检查。竞态测试需要 Linux C 编译器（GitHub Ubuntu runner 已提供）。
- [actions/setup-go](https://github.com/actions/setup-go) v6.5.0 固定至上游核实的完整 commit `924ae3a1cded613372ab5595356fb5720e22ba16`；Actions 保持只读 token、超时和原有并发策略。

## 目录

```text
cmd/server/              程序入口、信号处理和监听失败退出
internal/config/         环境变量读取与校验、固定 loopback 地址
internal/server/         版本化 Gin Router、HTTP 超时和请求排空退出
internal/database/       GORM/SQLite 连接与显式版本迁移
scripts/smoke_backend.py 已构建 Linux 程序的启动/退出验证
go.mod / go.sum          版本和校验和
Makefile                依赖、构建、测试、启动命令
.env.example            无凭据的配置说明模板
```

不预先创建业务分层空目录。后续根据实际功能引入 handler、service、repository 等包。

## Debian WSL 本地运行

所有命令从项目仓库根目录执行。本机 Go PATH 和 nvm 在交互式 Bash 中初始化，因此在 Codex 中使用：

```bash
cd /home/zhanglin/projects/pokemon-shiny-tracker
bash -ic 'pwd; command -v go; go version; go env GOOS GOARCH'
bash -ic 'make deps'
bash -ic 'make run'
```

应选中 `/usr/local/go/bin/go`，系统为 Linux。服务默认监听 `127.0.0.1:8080`，启动日志包含实际监听地址。另一终端可运行 `curl --noproxy "*" -i http://127.0.0.1:8080/api/v1/health`，预期为 **200** 和 `{"data":{"status":"ok"}}`；它只证明 HTTP 进程可响应，不证明数据库/业务健康。完整错误格式见 [API 契约](API.md)。

按 Ctrl+C 或发送 SIGTERM，服务最多等待 5 秒排空活动请求；正常退出码为 0，配置错误或端口占用退出码为 1。HTTP 请求头超时 5 秒，读/写超时 15 秒，空闲连接超时 60 秒。

如果交互 Bash 在自动化中无法工作，可只为当前命令显式配置环境，无需重装或修改 shell 文件：

```bash
cd /home/zhanglin/projects/pokemon-shiny-tracker
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
export NVM_DIR="$HOME/.nvm"
[ ! -s "$NVM_DIR/nvm.sh" ] || . "$NVM_DIR/nvm.sh"
go version
```

GitHub Ubuntu runner 的 PATH 由 setup-go 配置，不依赖本机 `.bashrc`，直接执行相同 Make 目标。

## 配置与个人数据

当前仅支持可选环境变量 `SHINY_HTTP_PORT`：不设置时为 8080；设置时须为 1–65535 的十进制整数，不接受空串、符号、空白或 host:port。监听主机固定 `127.0.0.1`，不提供远程访问开关。

```bash
bash -ic 'SHINY_HTTP_PORT=8081 go run ./cmd/server'
```

`.env.example` 只是说明模板，程序不会自动读取 `.env` 或认证文件。需要持久化本机环境时由调用方管理，真实配置/密钥不提交。程序没有认证密钥配置。T005 启动时在用户数据目录初始化 SQLite，测试仅在各自创建的临时目录运行；模型/迁移说明见 DATABASE.md。GORM 使用不输出绑定参数的日志配置。真实数据不进入源码或 CI。

## 构建、检查与启动验证

```bash
bash -ic 'go test ./...'
bash -ic 'go build ./...'
bash -ic 'make check'
bash -ic 'make smoke'
python3 scripts/validate_baseline.py
python3 -m unittest discover -s scripts -p 'test_*.py' -v
git diff --check
```

- `make deps`：下载锁定依赖并执行 `go mod verify`；首次需要联网，缓存就绪后运行程序本身不访问网络。
- `make build`：构建所有包，并产生被 Git 忽略的 `bin/shiny-tracker`。
- `make check`：格式检查、`go mod tidy -diff`、依赖校验、`go vet`、竞态测试和构建。
- `make smoke`：构建后测试非法配置、端口占用、真实 loopback health 200 和 JSON 404/405、SIGTERM 正常退出及端口释放；只使用临时端口，不接触个人数据。
- `make fmt`：主动格式化 Go 源文件；它会修改源文件，仅在开发时使用。

配置单元测试覆盖默认值、端口边界和非法输入。HTTP 测试覆盖健康/错误契约、404/405、内部错误/panic 脱敏、错误后继续服务、已提交响应不被追加、真实监听/错误和排空退出。GitHub `go-checks` 运行上述 Go 检查和真实进程验证，现有治理检查继续执行。

SQLite 行为由 T005 的真实数据库测试验证；Windows 原生启动和业务 CRUD 在相应 Task 验收。
