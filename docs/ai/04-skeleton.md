# v0.1 MVP System Skeleton

## 1. 模块拆分

- Web UI 模块
  - 提供群聊列表、群聊详情、AI 角色配置、基础模型 API 配置四类页面。
  - 只服务 v0.1 用户操作路径，不包含登录、主题、夜间模式、统计面板等非 P0 功能。

- HTTP API 模块
  - 承接网页表单和页面请求。
  - 暴露群聊、角色、模型配置、消息发送和历史读取所需的最小接口。

- Application Service 模块
  - 编排用户操作流程。
  - 负责创建群聊、添加角色、保存配置、发送消息、触发多 AI 回复。

- Domain 模块
  - 定义 v0.1 核心业务对象：群聊、AI 角色、消息、模型配置。
  - 保持业务规则集中，避免 UI 或存储层直接拼接核心逻辑。

- Persistence 模块
  - 持久化群聊、AI 角色、模型 API 配置和消息历史。
  - MVP 只需要单用户数据空间，不设计多租户和复杂权限模型。

- AI Client 模块
  - 封装真实模型调用路径。
  - 根据 AI 角色的人设、回复风格、模型选择和群聊上下文生成回复。

## 2. 目录结构

```text
.
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── chat_service.go
│   │   ├── role_service.go
│   │   ├── settings_service.go
│   │   └── message_service.go
│   ├── domain/
│   │   ├── chat.go
│   │   ├── role.go
│   │   ├── message.go
│   │   └── model_config.go
│   ├── http/
│   │   ├── routes.go
│   │   ├── handlers_chats.go
│   │   ├── handlers_roles.go
│   │   ├── handlers_settings.go
│   │   └── handlers_messages.go
│   ├── ai/
│   │   ├── client.go
│   │   ├── prompt.go
│   │   └── provider_openai_compatible.go
│   └── store/
│       ├── store.go
│       ├── sqlite.go
│       └── migrations.go
├── web/
│   ├── templates/
│   │   ├── layout.html
│   │   ├── chats_index.html
│   │   ├── chat_detail.html
│   │   ├── role_form.html
│   │   └── settings.html
│   └── static/
│       └── app.css
├── docs/
│   └── ai/
└── README.md
```

## 3. 各模块职责

- `cmd/server`
  - 启动 HTTP 服务。
  - 初始化存储、服务、路由和模板。

- `internal/http`
  - 处理页面请求和表单提交。
  - 将输入转换为 application service 调用。
  - 将结果渲染为网页。

- `internal/app`
  - 执行 MVP 主链路。
  - 保证发送消息时会保存用户消息，并为至少两个 AI 角色生成和保存回复。
  - 校验创建群聊、添加角色、模型配置、发送消息所需的最小输入。

- `internal/domain`
  - 定义核心实体和枚举。
  - 表达发送者类型、角色配置、模型配置、消息归属等基础规则。

- `internal/store`
  - 提供持久化接口和数据库实现。
  - 负责基础迁移、CRUD 和按群聊读取消息历史。

- `internal/ai`
  - 定义模型调用接口。
  - 组装角色提示词和上下文。
  - 隔离具体模型供应商，避免业务层依赖供应商 SDK 细节。

- `web/templates`
  - 提供 v0.1 所需页面。
  - 页面必须覆盖完整用户操作路径，不能只依赖接口调试。

## 4. 核心接口或 API

- 页面路由
  - `GET /`：跳转或展示群聊列表。
  - `GET /chats`：展示群聊列表和创建入口。
  - `POST /chats`：创建群聊。
  - `GET /chats/{chatID}`：展示群聊详情、消息历史、角色列表和发送入口。
  - `POST /chats/{chatID}/roles`：为群聊添加 AI 角色。
  - `GET /settings/model`：展示基础模型 API 配置。
  - `POST /settings/model`：保存基础模型 API 配置。
  - `POST /chats/{chatID}/messages`：发送用户消息并触发 AI 回复。

- 应用服务接口
  - `CreateChat(name) -> Chat`
  - `ListChats() -> []Chat`
  - `GetChat(chatID) -> ChatDetail`
  - `AddRole(chatID, name, persona, style, model) -> Role`
  - `SaveModelConfig(provider, baseURL, apiKey, model) -> ModelConfig`
  - `SendUserMessage(chatID, content) -> MessageResult`

- 存储接口
  - `CreateChat(chat)`
  - `ListChats()`
  - `GetChat(chatID)`
  - `CreateRole(role)`
  - `ListRoles(chatID)`
  - `SaveModelConfig(config)`
  - `GetModelConfig()`
  - `CreateMessage(message)`
  - `ListMessages(chatID)`

- AI 客户端接口
  - `GenerateReply(role, chat, messages, modelConfig, userMessage) -> AIReply`
  - AI 客户端必须走真实模型调用路径；模型服务失败时返回清晰错误，不用固定 mock 文本冒充成功。

## 5. 数据结构

- Chat
  - `id`
  - `name`
  - `created_at`
  - `updated_at`

- Role
  - `id`
  - `chat_id`
  - `name`
  - `persona`
  - `reply_style`
  - `model`
  - `created_at`
  - `updated_at`

- Message
  - `id`
  - `chat_id`
  - `sender_type`
  - `sender_name`
  - `role_id`
  - `content`
  - `created_at`

- ModelConfig
  - `id`
  - `provider`
  - `base_url`
  - `api_key`
  - `default_model`
  - `created_at`
  - `updated_at`

- MessageResult
  - `user_message`
  - `ai_messages`
  - `errors`

- SenderType
  - `user`
  - `ai`

## 6. 主链路时序

- 创建群聊
  - 用户打开群聊列表页。
  - 用户提交群聊名称。
  - HTTP Handler 调用 `CreateChat`。
  - Service 校验名称并写入 Store。
  - 页面跳转到群聊详情页。

- 添加 AI 角色
  - 用户在群聊详情页填写角色名称、人设、回复风格和模型。
  - HTTP Handler 调用 `AddRole`。
  - Service 校验角色属于当前群聊并写入 Store。
  - 页面刷新后展示角色列表。

- 配置模型 API
  - 用户进入基础模型 API 配置页。
  - 用户填写 provider、base URL、API key、默认模型。
  - HTTP Handler 调用 `SaveModelConfig`。
  - Service 保存配置。
  - 页面展示保存结果。

- 发送消息并生成 AI 回复
  - 用户在群聊详情页发送消息。
  - HTTP Handler 调用 `SendUserMessage`。
  - Service 保存用户消息。
  - Service 读取当前群聊、角色列表、历史消息和模型配置。
  - Service 校验至少存在两个 AI 角色。
  - Service 为至少两个 AI 角色分别调用 AI Client。
  - AI Client 根据角色人设、回复风格、模型选择和上下文请求真实模型。
  - Service 保存每个 AI 回复。
  - 页面重新渲染消息列表，展示用户消息和 AI 角色回复。

- 查看历史消息
  - 用户重新进入群聊详情页。
  - HTTP Handler 读取群聊详情、角色列表和消息历史。
  - 页面展示历史消息，并保留发送者身份。

## 7. 权限、隔离和边界

- v0.1 不实现用户登录和多用户账号隔离。
- v0.1 采用单用户数据空间，所有群聊、角色、配置和消息属于同一默认用户上下文。
- 所有数据访问必须通过 Store 接口，不让页面层直接访问数据库。
- AI Client 只能读取当前群聊所需的角色配置、模型配置和消息上下文。
- API Key 只用于服务端模型调用，不在消息页面展示明文。
- 不实现团队、组织、共享、角色权限、发言权限开关或复杂访问控制。
- 不实现文件上传、工具调用、Token 统计、主题引导、夜间模式或实时推送。
- 如果实现中需要引入非 P0 权限或隔离能力，必须停止并回到 `docs/ai/03-mvp-contract.md` 确认。

## 8. 测试切入点

- Service 层测试
  - 创建群聊时名称不能为空。
  - 添加角色时必须关联存在的群聊。
  - 发送消息前必须存在基础模型配置。
  - 发送消息前必须至少有两个 AI 角色。
  - 发送消息会保存用户消息和 AI 回复。

- Store 层测试
  - 群聊可以创建、读取和列表展示。
  - 角色可以按群聊保存和读取。
  - 模型配置可以保存和读取。
  - 消息可以按群聊按时间顺序读取。

- AI Client 边界测试
  - 能根据角色配置组装包含人设、回复风格和上下文的请求。
  - 模型调用失败时返回明确错误。
  - 不用固定 mock 文本标记 MVP 成功。

- HTTP/UI 验收测试
  - 用户可以通过网页创建群聊。
  - 用户可以通过网页添加两个 AI 角色。
  - 用户可以通过网页保存模型 API 配置。
  - 用户可以通过网页发送消息。
  - 页面展示用户消息和至少两个 AI 角色回复。
  - 刷新或重新进入群聊后仍显示历史消息。
