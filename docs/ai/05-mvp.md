# v0.1 MVP Implementation Record

## 本轮最小可运行闭环

用户打开 Gin 网页应用后，可以创建群聊、进入群聊详情、保存基础模型 API 配置、添加至少两个 AI 角色、发送用户消息、触发 AI 角色回复编排，并在群聊详情页回看保存的消息历史。

## MVP 合同对照

- 使用 Go + Gin，未替换技术栈。
- 使用 MySQL 作为持久化目标，包含迁移和 Store 实现。
- 使用 Redis 作为已确认基础设施，启动时执行连通检查；v0.1 不扩展 Redis 非 P0 功能。
- 提供网页端 UI，不是纯后端 demo。
- 提供表单操作路径，不是静态文案或只靠接口测试。
- AI 回复走 OpenAI-compatible `/chat/completions` 真实调用路径。
- 未实现登录、完整 API 管理、WebSocket、文件上传、工具调用、Token 统计、主题引导、夜间模式、角色互相辩论等非 P0 功能。

## 已实现

- Go module 和 Gin 服务入口。
- 本地启动脚本 `scripts/run-local.sh`，默认使用 8080，若端口占用会自动选择 8081-8090 的第一个空闲端口。
- 中文群聊列表页、创建群聊表单、群聊详情页。
- AI 角色添加表单，包含名称、人设、回复风格和模型；模型从设置中的支持模型列表下拉选择，不再手动输入。
- 基础模型 API 配置页，包含 provider、base URL、API key、支持模型列表、默认模型。
- 模型 API 设置页支持“检测连接并获取模型”，成功后从 OpenAI-compatible `/models` 自动填充支持模型列表。
- 默认模型改为从已获取的支持模型列表中下拉选择，不再手动输入。
- 发送消息失败时展示每个 AI 角色的具体失败原因，避免只显示笼统的回复数量不足。
- 系统状态页 `/health`，展示 MySQL 和 Redis 是否正常。
- 用户消息发送表单。
- MySQL 表结构迁移：`chats`、`roles`、`model_configs`、`messages`。
- `model_configs` 增加支持模型列表字段，并兼容已有表结构自动补列。
- 未提供 `MYSQL_DSN` 时，应用默认使用 `root/4399` 连接 `127.0.0.1:3306`，并自动创建 `ai_chat` 数据库。
- 未提供 Redis 配置时，应用默认使用 `127.0.0.1:6379` 和密码 `4399`。
- Store 层：群聊、角色、模型配置、消息的创建和读取。
- Service 层：创建群聊、添加角色、保存模型配置、发送消息并触发至少两个 AI 角色回复。
- AI Client：OpenAI-compatible Chat Completions 调用。
- 网页路由主路径测试。
- Service 层 MVP 规则测试。

## 已验证

- `go test -mod=mod ./...` 通过。
- `go build -mod=mod -buildvcs=false ./cmd/server` 通过。
- 用户确认 v0.1 主路径已成功跑通。
- 用户确认发送消息后可以得到 AI 回复。
- 用户确认当前主要体验问题是发送后页面刷新卡顿，期望后续改为类似聊天群的实时体验。
- 已验证模型配置必须包含支持模型列表，默认模型必须属于该列表。
- 已验证添加 AI 角色时只能选择设置中存在的模型。
- 已验证模型设置检测接口会调用模型列表获取流程，并把模型列表带回设置页面。
- 已验证 AI 回复不足两个时，错误信息包含具体角色失败原因。
- 已验证 `/health` 路由可渲染 MySQL/Redis 状态。
- 已验证代码启动路径会使用 `root@tcp(127.0.0.1:3306)/ai_chat` 并先执行自动建库逻辑。
- Service 层测试验证：
  - 发送消息前少于两个 AI 角色会被阻止。
  - 发送消息后会保存 1 条用户消息和 2 条 AI 回复。
- HTTP/UI 路由测试验证：
  - `GET /chats` 可访问。
  - `POST /chats` 可创建群聊并跳转详情页。
  - `POST /settings/model` 可保存基础模型 API 配置。
  - `POST /chats/{chatID}/roles` 可添加两个 AI 角色。
  - `POST /chats/{chatID}/messages` 可发送消息并触发回复编排。
  - 再次 `GET /chats/{chatID}` 能看到用户消息和两个 AI 角色回复。

## 未验证

- 真实 MySQL 服务已由用户本机启动流程间接验证可用，但本执行环境仍无法直接连接复核。
- 真实 Redis 服务已由用户本机启动流程间接验证可用，但本执行环境仍无法直接连接复核。
- 未通过浏览器或 Playwright 完成人工点击验证。
- 未由本执行环境在用户真实 API 上复核 `/models` 拉取结果和真实聊天回复。
- 未由本执行环境验证真实刷新页面后的历史消息回看；目前用户主路径已成功，本地自动测试覆盖同一进程内主路径。

## 阻塞

- 当前沙箱环境直接访问本地 MySQL 失败：`ERROR 2004 (HY000): Can't create TCP/IP socket (1)`。
- 当前沙箱环境直接访问本地 Redis 失败：`Could not connect to Redis at 127.0.0.1:6379: Can't create socket: Operation not permitted`。
- 已按规则请求提升权限验证 MySQL/Redis TCP 连接，但审批服务返回 503，命令未获准执行。
- 临时构建产物 `server` 已生成；删除命令审批失败，因此文件仍在工作区，但已加入 `.gitignore`。

## 下一步

v0.1 MVP 完成。下一步进入 v0.2 可用版：优先处理聊天群式实时体验、AI 角色删除、群聊删除，以及更清晰的前端交互。
