# Build Log

## 2026-05-02 v1.0 File Upload Slice

### 最小可运行闭环

用户在群聊页面上传受支持文本文件，系统保存文件元数据和可读文本，聊天页面展示文件列表；之后用户发送消息时，普通 AI 回复和 AI 互评都能收到文件内容上下文。

### 实现步骤

- 新增 `domain.ChatFile` 和 `ChatDetail.Files`。
- 新增 Store 接口：`CreateChatFile`、`ListChatFiles`。
- 新增 MySQL 表 `chat_files`，按 `user_id` 和 `chat_id` 关联文件。
- 新增服务方法 `AddChatFile`，校验聊天归属、文件元数据和可读文本。
- 更新 `GetChat`，返回当前群聊的文件列表。
- 新增 HTTP 路由 `POST /chats/:chatID/files`。
- 新增文件上传校验：`txt`、`md`、`json`、`csv`、`log`、`docx`、`pdf`，最大 10MB。
- 新增 `.docx` 文本提取：读取 Office Open XML 中的正文、页眉和页脚 XML。
- 新增基础 PDF 文本提取：支持常见可复制文本 PDF 的 literal/hex 字符串与 FlateDecode stream；扫描版 PDF 不支持。
- 扩展聊天页 `accept` 类型，让本地文件选择器可以选择 `.docx` 和 `.pdf`。
- 新增本地拖拽上传区域，文件可通过点击选择或拖到上传区域提交。
- 聊天文件默认保存到 `data/chat-files`，避免默认暴露在 `/uploads` 静态目录。
- 聊天页面新增“文件资料”面板和文件列表。
- AI 输入结构新增 `Files`，OpenAI-compatible 客户端把文件上下文追加到 system prompt。
- 文件上下文限制为最多 12000 rune，避免 prompt 无边界增长。

### 非本轮需求

- 文件删除、下载、预览、OCR、复杂 PDF 排版解析、向量检索和长期知识库没有实现，继续留在后续 backlog 或后续 v1.0 切片中处理。

## 2026-05-02 v1.0 Controlled Tools Slice

### 最小可运行闭环

用户在群聊页面手动选择一个内置受控工具，提交输入后系统执行工具，把结果保存为系统消息，并在工具执行记录中显示成功或失败状态。

### 实现步骤

- 新增 `domain.ToolExecution` 和 `ChatDetail.Tools`。
- 新增 Store 接口：`CreateToolExecution`、`ListToolExecutions`。
- 新增 MySQL 表 `tool_executions`，按 `user_id` 和 `chat_id` 关联工具记录。
- 新增服务方法 `ListTools` 和 `RunTool`。
- 实现内置工具白名单：`current_time`、`text_stats`、`calculator`。
- 计算器只解析数字、括号和 `+ - * /`，不执行任何代码。
- 工具执行成功和失败都会生成系统消息。
- 新增 HTTP 路由 `POST /chats/:chatID/tools`。
- 聊天页面新增“受控工具”面板和最近工具执行记录。

### 非本轮需求

- 不做模型自动 tool calling、shell 执行、外网请求、文件系统工具、插件市场和复杂审批流。

## 2026-05-03 AI Review Async Toggle Bug Fix

### 最小可运行闭环

用户在群聊详情页点击 AI 互评开关时，支持 JavaScript 的浏览器通过异步请求切换状态，页面不整页刷新；无 JavaScript 时保留原普通表单重定向兜底。

### 实现步骤

- `/chats/:chatID/ai-review` 在 JSON 请求下返回当前开关状态。
- `chat.js` 拦截 AI 互评开关表单，使用 `fetch` 提交。
- 前端局部更新消息区状态、右侧菜单状态、按钮文本、隐藏字段和 `data-ai-review-enabled`。
- 保留非 JSON 表单请求的 `302` 重定向行为。
- 增加 HTTP 回归测试覆盖异步标记、普通表单兜底和 JSON 返回。

## 2026-05-03 v1.0 Multi-Provider/Multi-Model Routing Slice

### 最小可运行闭环

用户保存多个模型 API 配置后，可以在设置页和角色卡片中看到明确的路由编号、供应商、配置名称和模型；普通回复和 AI 互评都按每个角色绑定的路由执行。

### 实现步骤

- 复用现有 `model_configs`、`roles.model_config_id` 和角色模型选择，不新增表。
- 增加模板函数 `configByID`，让聊天页可按角色 `model_config_id` 查找配置展示信息。
- 设置页的已保存配置展示路由编号、供应商、默认模型、模型数量和 Base URL。
- 角色新增/编辑下拉项展示路由编号、配置名称和模型。
- 角色卡片展示当前路由编号、配置名称、供应商和模型。
- 增加服务层测试，验证普通回复按角色绑定的模型配置调用。
- 增加服务层测试，验证 AI 互评按角色绑定的模型配置调用。
- 更新 HTTP 测试，验证设置页和聊天页能检查路由信息。

### 非本轮需求

- 不做自动故障转移、负载均衡、价格路由、自动模型选择或新供应商 SDK。

## 2026-05-03 v1.0 Deployment Documentation Slice

### 最小可运行闭环

新部署者可以从 README 进入部署文档，并根据文档完成当前应用的自托管部署准备：构建二进制、配置 MySQL/Redis/环境变量、准备持久化目录、放到进程管理器后面运行、通过反向代理/TLS 暴露，并知道如何做启动验证、备份和回滚。

### 实现步骤

- 新增 `docs/ai/deployment.md`。
- 记录目标部署形态：Linux/VM、Go server、MySQL、Redis、反向代理和持久化目录。
- 记录二进制构建命令。
- 记录生产环境变量示例，包括 `MYSQL_DSN`、Redis、模板/静态资源、上传目录、聊天文件目录和模型 API 超时配置。
- 记录 `UPLOAD_DIR`、`CHAT_FILE_DIR` 和 MySQL 的持久化/备份要求。
- 记录 systemd 示例。
- 记录反向代理/TLS 和上传 body size 注意事项。
- 记录启动验证、备份、恢复和回滚步骤。
- README 与 `developer-settings.md` 链接部署文档。

### 非本轮需求

- 不实际部署生产环境。
- 不申请域名或 TLS 证书。
- 不新增 Docker/Kubernetes/云平台自动化。
- 不实现 observability 或 CI。

## 2026-05-03 v1.0 Observability/Logging/Error Strategy Slice

### 最小可运行闭环

关键错误路径会输出明确日志标签，运营者可以从进程日志中定位 HTTP 错误、聊天页操作失败、异步 JSON 错误和异步 AI 回复失败；同时有策略文档说明当前 baseline、敏感信息边界和后续增强。

### 实现步骤

- 新增 `docs/ai/observability.md`。
- HTTP `renderError` 输出 `http_error` 日志。
- 聊天页 action 失败输出 `chat_action_error` 日志，包含 action、method、path、chat ID、status 和 error。
- AI 互评切换、主题保存、文件上传、工具运行、异步发送、消息更新等错误路径接入聊天 action 日志。
- 异步 AI 回复失败输出 `async_ai_reply_error` 日志。
- 新增 HTTP 测试验证聊天 action error 日志输出。

### 非本轮需求

- 不接入外部错误报告 SaaS。
- 不实现 metrics、alerting、tracing 或 request ID。
- 不替换 Gin logger。

## 2026-05-03 v1.0 CI Checks Slice

### 最小可运行闭环

开发者可以运行 `sh scripts/ci-check.sh`，GitHub Actions 也有等价的 test/build 工作流；检查不需要 MySQL、Redis、模型 API、浏览器或部署密钥。

### 实现步骤

- 新增 `scripts/ci-check.sh`。
- 脚本运行 `go test -mod=mod ./...`。
- 脚本运行 `go build -mod=mod -buildvcs=false ./cmd/server`。
- 新增 `.github/workflows/ci.yml`。
- 工作流在 push 和 pull request 上运行。
- 工作流使用 `go-version-file: go.mod`。
- 新增 `docs/ai/ci.md`。
- README 增加本地检查命令。

### 非本轮需求

- 不做浏览器 E2E。
- 不接 MySQL/Redis/模型 API。
- 不部署。
- 不使用 secrets。

## 2026-05-03 AI Review Natural-Reply Prompt Optimization

### 最小可运行闭环

开启 AI 互评后，额外生成的 AI 回复仍走现有异步追加、角色路由、文件上下文和消息保存流程，但 prompt 不再引导模型写正式“补充/反驳报告”，而是引导它像群聊成员一样自然接住某个 AI 角色刚说过的话。

### 实现步骤

- 将 `BuildReviewSystemPrompt` 的定位从“参与互评”调整为“继续接话，不是在写互评报告”。
- 主题约束从“补充或反驳必须服务主题”改为“接话必须服务主题，发散时自然拉回主题”。
- 要求模型选择一个最值得回应的观点，而不是逐条点评所有回复。
- 允许的互动方式明确为认同、补充、追问、指出风险或温和反驳。
- 要求自然提到被回应角色名，避免总结全场。
- 控制输出为 1 到 3 段短段落，并减少“首先/其次/综上”等报告式表达。
- 将 `BuildReviewConversation` 的任务语气同步改为“用群聊里的自然语气接话”。
- 增加单元测试覆盖互评 prompt 的自然接话约束和 conversation 文案。

### 非本轮需求

- 不改变互评触发时机、最多互评条数、角色选择算法、数据库结构、前端展示或轮询策略。
- 不接入真实模型自动评分；真实输出质感由用户在浏览器中用已配置模型验收。

## 2026-05-03 Selective AI Participation Optimization

### 最小可运行闭环

用户发送消息后，系统不再让所有可发言 AI 角色每轮都固定回答；AI 互评也不再每次固定追加两个回复。当前最小机制保留至少两个 AI 回复的 MVP 主路径，同时让群聊更接近“有人接话、有人沉默”的真实聊天节奏。

### 实现步骤

- `generateAIReplies` 改为调用选择函数；当可发言角色超过两个时，每轮稳定选择两个首轮发言角色。
- 角色选择使用消息 ID 和内容生成稳定索引，避免同一条消息在重试时随机变化。
- `appendAIReviews` 增加触发判断：
  - AI 互评开关必须开启。
  - 第一轮至少有两个 AI 回复。
  - 用户消息不能过短，短消息不自动互评。
- 互评角色选择从固定前两个角色改为最多一个角色跟进；优先选择没有参与第一轮回复的角色，如果没有候选则从现有角色中稳定选择一个。
- 前端消息轮询从“等待所有角色 + 固定互评数量”改为“等待至少两个核心 AI 回复，再经过短暂静默后停止”；开启互评时静默等待略长，以便可选互评追加。
- 更新服务层和 HTTP 测试，覆盖短消息跳过互评、长消息最多一个互评、选择性首轮回复和异步追加路径。

### 非本轮需求

- 不新增 UI 配置项。
- 不新增数据库字段。
- 不让模型先判断每个角色是否应该发言。
- 不实现多轮持续辩论或复杂发言调度。

## 2026-05-03 Faster AI Reply and Status Text Optimization

### 最小可运行闭环

用户发送消息后，首轮选中的 AI 角色并发请求模型，减少等待多个模型串行返回的总时间；聊天底部只显示简洁的“AI 正在回复...”，不再暴露“随后进行互评...”这种流程感文案。

### 实现步骤

- `generateAIReplies` 先计算本轮首轮发言角色。
- 为每个首轮角色启动 goroutine 并发调用 `GenerateReply`。
- 等待所有首轮模型调用完成后，按选择顺序保存 AI 消息。
- token usage 仍在消息保存后记录，保证有 message ID。
- AI review 仍在首轮消息保存完成后按现有选择性机制触发。
- 前端发送成功后的状态文案改为固定 `AI 正在回复...`。
- 增加服务层并发回归测试，验证两个首轮 AI 调用都会在释放任意一个响应前开始。

### 非本轮需求

- 不改变模型供应商超时、重试或路由策略。
- 不并发写入 store。
- 不实现流式输出。
- 不新增 UI 配置项。

## 2026-05-03 AI Reply Strategy Rollback and Static Cache-Busting Fix

### 最小可运行闭环

用户发送消息后，所有允许发言的 AI 角色都应完成首轮回复；首轮模型调用仍并发加速；页面不会因为等待策略错误而只出现两个回复后卡住；聊天页加载带版本参数的 `chat.js`，避免浏览器缓存旧状态文案。

### 实现步骤

- `selectFirstRoundRoles` 回滚为返回所有允许发言角色。
- `generateAIReplies` 继续并发调用所有首轮角色的 `GenerateReply`。
- `minimumAIReplies` 改为当前 `roleCount`，前端轮询不再两个回复后进入静默停止条件。
- `chat_detail.html` 为 `app.css`、`theme.js`、`chat.js` 增加 `?v=20260503a`。
- HTTP 测试增加聊天页静态资源带版本参数断言。

### 非本轮需求

- 不重新引入“只选两个角色回答”策略。
- 不改变 AI 互评最多一条的限制。
- 不实现流式输出。

## 2026-05-04 AI Review Trigger Simplification

### 最小可运行闭环

用户开启 AI 互评后，只要首轮至少两个 AI 回复成功，就追加一条互评回复；短消息也会触发互评，方便用户确认功能是否正常。

### 实现步骤

- 移除 `shouldAppendAIReview` 中的 18 rune 短消息门槛。
- 保留 AI 互评开关判断。
- 保留首轮至少两个 AI 回复成功的判断。
- 保留最多一条互评回复的选择逻辑。
- 更新服务层测试，覆盖短消息也会触发互评。
- 更新 HTTP 异步测试，覆盖普通短消息也能追加 `review from`。

### 非本轮需求

- 不恢复固定两条互评。
- 不新增 UI 配置项。
- 不引入模型判断是否互评。

## 2026-05-04 AI Review Polling Fix

### 最小可运行闭环

用户开启 AI 互评后，当前页面轮询会等待首轮所有可发言 AI 回复和额外 1 条互评回复，不会在只收到首轮回复后提前停止。

### 实现步骤

- `minimumAIReplies` 在 AI 互评开启时从 `roleCount` 改为 `roleCount + 1`。
- 保持 AI 互评关闭时等待 `roleCount`。
- 保持互评最多追加 1 条。
- 将聊天页静态资源版本从 `20260503a` 升级到 `20260504a`。
- 更新 HTTP 测试断言聊天页加载 `chat.js?v=20260504a`。

### 非本轮需求

- 不增加互评条数。
- 不改变互评生成 prompt。
- 不实现 WebSocket 或流式输出。
