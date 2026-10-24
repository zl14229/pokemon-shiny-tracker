# M0 API 契约

后端默认 `http://127.0.0.1:8080`，API 前缀 `/api/v1`。当前只提供进程健康检查，不访问 SQLite 或外部资料源。

| 请求 | HTTP | JSON |
|---|---|---|
| `GET /api/v1/health` | 200 | `{"data":{"status":"ok"}}` |
| 未注册路径 | 404 | `{"error":{"code":"not_found","message":"请求的接口不存在"}}` |
| 已注册路径的不支持方法 | 405，`Allow: GET`（health） | `{"error":{"code":"method_not_allowed","message":"请求方法不受支持"}}` |
| 尚未发送响应时的内部错误或 panic | 500 | `{"error":{"code":"internal_error","message":"服务暂时不可用，请稍后重试"}}` |

响应内容类型为 `application/json; charset=utf-8`。客户端以 HTTP 状态和 `error.code` 判别错误，`message` 用于提示。内部错误、panic 原值和堆栈不写入响应。服务端只记录固定的失败提示，不记录原始错误或请求输入。

健康结果仅表示 HTTP 进程可响应，不代表数据库、资料覆盖率或业务功能已就绪。`/health`、`/api/v2/health` 和 `/api/v1/health/` 不作为别名。HEAD 不隐式映射为 GET；HTTP 协议会省略 HEAD 响应体。

错误中间件处理 `c.Error` 和 panic。若 handler 已提交响应，则终止后续处理但不追加第二段 JSON，也无法撤销已发出的状态/内容；后续 handler 应先验证与完成可能失败的操作，再写响应。测试专用失败路由只在测试中注册，实际程序不提供故障触发入口。

目前不启用 CORS、认证、数据库检查或业务端点。T004 前端使用 `/api/v1` 同源代理调用这些端点，健康 DTO 在运行时校验；非 JSON、格式不符、非 2xx、网络/超时均不得被视为健康。完整客户端行为见前端开发文档。
