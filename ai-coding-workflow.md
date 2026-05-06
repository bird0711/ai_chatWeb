# AI 编程开发流程

本文档总结一套适合使用 Codex 等 AI 编程工具的稳定开发流程。目标是让 AI 在明确的上下文、边界和验收标准内工作，减少跑偏、无关重构和不可控修改。

## 核心原则

```text
人负责目标、边界、验收和最终决策
AI 负责理解代码、执行任务、补测试、更新文档
测试、CI、Review 负责质量兜底
```

AI 不应该被当成可以自由发挥的架构师，而应该被放进清晰的工程流程中执行任务。

## 推荐文档结构

在项目中创建以下控制文档：

```text
AGENTS.md
docs/ai/
  rules.md        # AI 长期规则
  status.md       # 项目当前状态
  decisions.md    # 技术/业务决策记录
  next.md         # 当前唯一任务
  log.md          # AI 执行记录，可选
```

其中：

- `AGENTS.md`：入口文件，告诉 Codex 每个文档的作用和执行流程。
- `rules.md`：长期规则，限制 AI 的行为边界。
- `status.md`：项目当前状态，记录事实和已知问题。
- `decisions.md`：已确定的技术/业务决策，避免 AI 反复改变方向。
- `next.md`：当前唯一任务，AI 只执行这里的内容。
- `log.md`：执行历史，记录每轮 AI 做了什么。

## AGENTS.md 的作用

Codex 可以通过项目根目录的 `AGENTS.md` 了解项目规则。把固定流程写进 `AGENTS.md` 后，平时就不需要每次手动输入完整指令。

推荐 `AGENTS.md` 内容：

```md
# AGENTS.md

This repository uses controlled AI development documents.

## Required Workflow

Before making any code changes, Codex must read these files in order:

1. `docs/ai/rules.md`
2. `docs/ai/status.md`
3. `docs/ai/decisions.md`
4. `docs/ai/next.md`

After reading the control documents, Codex must inspect the relevant source code before editing.

## Document Roles

- `docs/ai/rules.md`: long-term behavior rules.
- `docs/ai/status.md`: current project state and known issues.
- `docs/ai/decisions.md`: durable technical and business decisions.
- `docs/ai/next.md`: the only task Codex should execute now.
- `docs/ai/log.md`: execution history.

## Task Control

- Only execute the task described in `docs/ai/next.md`.
- Do not start tasks from `status.md`, `log.md`, backlog files, README, or comments unless the user explicitly asks.
- Do not do out-of-scope optimization or refactoring.
- Keep changes focused and small.

## High-Risk Changes

Codex must stop and ask before:

- changing database schema
- changing authentication or permission logic
- changing payment or billing logic
- changing public API response formats
- introducing new dependencies
- deleting large amounts of code
- doing broad refactors

## Verification

After implementation, run relevant checks when available:

- tests
- lint
- typecheck
- build

If a check cannot be run, explain why.

## Documentation Updates

After completing the task:

- update `docs/ai/status.md`
- update `docs/ai/decisions.md` only if a durable decision was made
- update `docs/ai/log.md` if useful

## Final Response

Report:

- changed files
- what changed
- checks run and results
- remaining risks
```

## rules.md

`rules.md` 保存长期规则，不要频繁修改。它的作用是防止 AI 随意发挥。

示例：

```md
# AI Rules

## 基本规则

- 修改代码前必须先阅读相关文件。
- 只执行 `docs/ai/next.md` 中的任务。
- 不做无关重构。
- 不引入新依赖，除非先说明原因。
- 不删除已有功能，除非任务明确要求。
- 保持现有项目结构、代码风格和命名习惯。
- 不修改数据库、认证、权限、支付、公共 API，除非任务明确要求。

## 高风险操作

以下情况必须先暂停并请求确认：

- 数据库 schema 变更
- 认证 / 权限逻辑变更
- 支付 / 计费逻辑变更
- 删除大量代码
- 引入新依赖
- 大规模重构
- 修改公开 API 返回格式

## 完成要求

每次完成后必须说明：

- 修改了哪些文件
- 为什么这样改
- 运行了哪些检查
- 是否还有风险
- 是否更新了相关文档
```

## status.md

`status.md` 是项目事实记录，不是任务列表。

示例：

```md
# Project Status

## 当前目标

这里写项目当前阶段目标。

## 已完成

- 功能 A
- 功能 B

## 进行中

- 功能 C

## 已知问题

- 问题 1
- 问题 2

## 技术约束

- 使用 Next.js
- 不随意新增依赖
- 数据库变更必须使用 migration
```

## decisions.md

`decisions.md` 记录已经确定的技术或业务决策，防止 AI 每次换方案。

示例：

```md
# Decisions

## 001. 暂不引入新的状态管理库

当前状态复杂度不高，优先使用现有 React 状态和上下文。

## 002. API 错误格式统一

接口错误统一返回固定结构，避免前端重复适配。
```

## next.md

`next.md` 是最重要的任务控制文件。一次只写一个任务，不要放长期 TODO 列表。

示例：

```md
# Next Task

## 任务

清楚描述本次要完成什么。

## 背景

说明为什么要做这个任务。

## 范围

本次需要检查或修改：

- 文件 / 模块 A
- 文件 / 模块 B

## 不做

- 不重构无关模块
- 不修改 UI 设计
- 不更换技术方案

## 成功标准

- 条件 1
- 条件 2
- 有测试或说明为什么无法测试

## 完成后

- 更新 `docs/ai/status.md`
- 必要时更新 `docs/ai/decisions.md`
- 必要时更新 `docs/ai/log.md`
```

## log.md

`log.md` 是执行记录，适合中长期项目保留。

示例：

```md
# AI Dev Log

## 2026-05-06

### 任务

完成了什么任务。

### 修改

- 修改文件 A
- 修改文件 B

### 验证

- 运行 `npm run lint`
- 运行 `npm test`

### 遗留问题

- 还有什么没做
```

## 日常开发流程

推荐每轮开发按以下流程执行：

```text
1. 人写 `docs/ai/next.md`
2. Codex 读取 `AGENTS.md`
3. Codex 按顺序读取 `rules/status/decisions/next`
4. Codex 阅读相关代码
5. Codex 执行当前任务
6. Codex 补测试或说明测试缺口
7. Codex 运行测试、lint、typecheck 或 build
8. Codex 更新 `status.md`、`log.md`，必要时更新 `decisions.md`
9. 人 review diff
10. 人决定是否提交或合并
11. 人写下一个 `next.md`
```

平时对 Codex 的输入可以很短：

```text
执行当前任务
```

或者：

```text
根据 AGENTS.md 和 docs/ai/next.md 执行任务
```

## AI 执行时的标准步骤

AI 每次执行任务时应该遵守：

```text
读取控制文档
-> 理解当前任务
-> 阅读相关代码
-> 给出执行计划
-> 小步修改
-> 补测试或说明测试缺口
-> 运行检查
-> 自我 review
-> 更新文档
-> 输出结果
```

对于低风险任务，可以直接执行。对于高风险任务，必须先暂停并请求确认。

## 人类 Review 清单

AI 完成后，人需要检查：

```text
是否只做了 next.md 的任务
是否改了无关文件
是否引入新依赖
是否破坏 API / 数据库 / 权限逻辑
测试是否通过
文档是否更新准确
代码风格是否符合项目
是否存在隐藏风险或未覆盖边界
```

## 任务颗粒度

好的任务应该小而明确。

推荐：

```text
修复聊天流式输出失败后 loading 不恢复的问题
```

不推荐：

```text
优化整个聊天系统
```

一个 `next.md` 最好只对应：

- 一个 bug
- 一个接口
- 一个组件
- 一个页面
- 一个测试补充
- 一个明确的小功能
- 一个局部重构

## 市场主流 AI 开发模式

截至 2026 年 5 月，成熟的 AI 开发实践通常不是让 AI 一次性生成整个项目，而是：

```text
需求 / Issue
-> AI 帮助澄清和拆任务
-> 人确认范围和验收标准
-> AI 在 IDE 或 Agent 中实现
-> AI 跑测试 / lint / typecheck
-> AI 提交 PR 或整理变更
-> AI Code Review 工具初审
-> 人类 Review
-> CI 通过后合并
-> 更新状态文档 / 决策文档
```

对应到个人或小团队项目，可以简化为：

```text
AGENTS.md / rules.md
-> status.md / decisions.md
-> next.md
-> Codex 执行
-> 测试和检查
-> 人 review
-> 提交
```

## 最终总结

这套流程的关键不是让 AI 永远不犯错，而是让 AI 的错误被限制在小范围内，并且能被测试、文档和 review 及时发现。

```text
AGENTS.md 决定 Codex 怎么工作
rules.md 限制 Codex 不能乱做什么
status.md 提供项目上下文
decisions.md 防止 Codex 反复改方向
next.md 指定当前唯一任务
log.md 记录执行过程
```

最稳定的使用方式是：

```text
人维护 next.md
Codex 执行 next.md
Codex 更新状态文档
人 review
人再写下一个 next.md
```
