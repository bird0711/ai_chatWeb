# Current Execution Contract

## 本轮目标

完成 v1.0/P3 的 CI 检查切片：提供一个无需外部服务、无需密钥、可在本地和 GitHub Actions 运行的最小质量门槛。

## 本轮范围

- 新增本地 CI 检查脚本。
- 新增 GitHub Actions 工作流。
- 新增 CI 文档。
- README 链接本地检查命令。
- CI gate 只运行自动测试和构建。

## 必须实现的用户路径

1. 开发者在本地运行 `sh scripts/ci-check.sh`。
2. 脚本运行 `go test -mod=mod ./...`。
3. 脚本运行 `go build -mod=mod -buildvcs=false ./cmd/server`。
4. GitHub Actions 在 push 或 pull request 时运行同样的 test/build gate。

## 必须实现的核心能力

- 本地可重复：同一命令可在开发机运行。
- CI 可执行：工作流不需要 secrets、MySQL、Redis 或模型 API。
- 主流程保护：测试和服务构建作为最低质量门槛。
- 文档清晰：贡献者知道如何运行检查。

## 必须验证的行为

- `scripts/ci-check.sh` 存在并可运行。
- `.github/workflows/ci.yml` 存在。
- `docs/ai/ci.md` 存在。
- README 包含本地检查命令。
- `go test -mod=mod ./...` 通过。
- `go build -mod=mod -buildvcs=false ./cmd/server` 通过。

## 明确不实现的内容

- 不运行浏览器 E2E。
- 不运行 MySQL/Redis 集成测试。
- 不调用真实模型 API。
- 不部署。
- 不使用 secrets。

## 技术约束

- CI 不依赖本机服务、数据库、Redis、浏览器或外网模型 API。
- 工作流不执行 push、release 或 deployment。
- 本地脚本只执行测试和构建。

## 验收清单

- [x] 新增本地 CI 检查脚本。
- [x] 新增 GitHub Actions 工作流。
- [x] 新增 CI 文档。
- [x] README 记录本地检查命令。
- [ ] 本地 CI 脚本执行通过。
- [ ] 自动测试通过。
- [ ] 构建通过。

## 停止条件

- 本地 CI 脚本无法通过且无法自行修复。
- GitHub Actions 需要 secrets 或外部服务。
- CI 范围扩展到部署、浏览器 E2E 或真实模型 API。
