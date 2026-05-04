# Risks

## 2026-05-02 v1.0 File Upload Slice

- 当前支持纯文本类文件、`.docx` 和基础文本型 PDF。图片、扫描件、音视频和复杂二进制文件无法分析。
- 扫描版 PDF 没有 OCR，无法提取图片里的文字。
- PDF 文本提取是无外部依赖的基础实现，复杂编码、复杂排版或特殊字体 PDF 可能提取不完整。
- 文件内容通过 system prompt 注入，适合小文件和早期闭环；大文件、多文件检索和精确引用需要后续向量检索或分片策略。
- 文件内容最多注入 12000 rune，超出部分会被截断，AI 可能无法看到完整文件。
- 已上传文件当前没有删除 UI；用户误传文件时需要后续补文件管理能力。
- 聊天分析文件默认保存到 `data/chat-files`，不公开静态服务；部署时仍需要按环境配置持久化目录和备份策略。

## 2026-05-02 v1.0 Controlled Tools Slice

- 当前工具调用是用户手动触发，不是模型自动 function calling。
- 工具范围刻意限制为内置白名单，不支持 shell、外网请求或文件系统访问。
- 计算器只支持基础四则表达式，不支持函数、变量或高精度数学。
- 工具执行记录当前只展示最近 20 条。

## 2026-05-03 AI Review Async Toggle Bug Fix

- 当前环境没有浏览器，页面不刷新的视觉行为依赖用户本机浏览器复测。
- 保留无 JavaScript 表单兜底，因此禁用 JavaScript 时仍会整页重定向，这是刻意兼容行为。

## 2026-05-03 v1.0 Multi-Provider/Multi-Model Routing Slice

- 当前切片是路由可见性和绑定执行的最小闭环，不包含自动故障转移。
- 所有供应商仍通过 OpenAI-compatible 协议调用；不兼容该协议的供应商仍不能直接接入。
- 路由选择仍由用户在角色配置中手动选择，不做自动模型选择或成本优化。
- API Key 仍按现有本地数据库路径保存；生产级加密、密钥轮换和密钥审计仍需后续部署/安全切片处理。

## 2026-05-03 v1.0 Deployment Documentation Slice

- 文档只覆盖通用自托管部署形态，没有针对具体云厂商或容器平台做自动化。
- systemd 示例需要部署者按自己的路径、用户、权限和服务依赖调整。
- 当前生产级日志、监控、告警和错误报告仍待 observability 切片。
- 当前 CI 仍待后续切片。
- API Key 生产级加密、轮换和审计没有在本切片实现。

## 2026-05-03 v1.0 Observability/Logging/Error Strategy Slice

- 当前日志是文本日志，不是结构化 JSON。
- 当前没有 request ID 或跨异步 worker 的 correlation ID。
- 当前没有 metrics、alerting 或外部错误报告集成。
- 日志记录高层错误文本；如果底层 provider 错误包含敏感信息，后续仍需要更严格的错误清洗策略。

## 2026-05-03 v1.0 CI Checks Slice

- 当前 CI 只覆盖 Go 测试和服务构建，不覆盖浏览器 E2E。
- 当前 CI 不启动 MySQL、Redis 或真实模型 API。
- GitHub hosted runner 是否支持 `go.mod` 中的 Go 版本取决于 runner/toolchain availability。
