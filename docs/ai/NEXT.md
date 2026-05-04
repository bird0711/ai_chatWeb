# AI Coding Next Step

## Unique Recommended Next Step

User acceptance checks for v1.0 minimum closure, all-role first-round replies, status text behavior, and reliable AI review visibility.

## Input Documents

- `docs/ai/STATUS.md`
- `docs/ai/BACKLOG.md`
- `docs/ai/DECISIONS.md`
- `docs/ai/developer-settings.md`
- `.env.example`
- `internal/ai/prompt.go`

## Current Result

- User reported on the host machine that the current model/API functionality is normal.
- The AI review toggle page-refresh regression has been fixed.
- AI review toggle now supports JSON and frontend `fetch` handling while preserving the original form redirect fallback.
- Automated tests and build pass after the AI review no-refresh fix.
- Running-service HTTP verification confirmed JSON AI review toggling returns `200 OK` instead of `302 Found`.
- v1.0 file upload first slice is implemented and code-verified.
- v1.0 controlled tools first slice is implemented and code-verified; HTTP tool execution was verified through the running service.
- v1.0 multi-provider/multi-model routing minimum closure is implemented.
- Route metadata is visible on settings and role cards.
- Service tests prove normal replies and AI review use each role's selected route.
- Automated tests and build pass.
- v1.0 deployment documentation slice is implemented.
- `docs/ai/deployment.md` exists and is linked from README and developer settings.
- Automated tests and build pass after deployment documentation changes.
- v1.0 observability/logging/error-reporting slice is implemented.
- `docs/ai/observability.md` exists.
- Key local error paths now emit process log tags.
- Automated tests and build pass after observability changes.
- Final v1.0 roadmap review is complete.
- All v1.0 roadmap implementation/documentation slices have current minimum closure evidence.
- `sh scripts/ci-check.sh` passes.
- AI review prompt has been optimized as a prompt-only quality slice:
  - AI review is framed as natural group-chat follow-up, not a formal review report.
  - The role is asked to choose one most worthwhile point to respond to.
  - The role can agree, add, ask a follow-up, point out risk, or gently challenge.
  - The role should avoid summarizing the whole discussion and report-like wording such as “首先/其次/综上”.
- Automated tests and build pass after the AI review prompt optimization.
- Selective AI participation has been implemented after user feedback that every AI role always answering and reviewing still felt mechanical:
  - when more than two roles can speak, each user message selects two first-round speakers instead of calling every role.
  - AI review is skipped for very short messages.
  - when AI review runs, it adds at most one follow-up reply instead of two fixed review replies.
  - frontend polling now stops after the minimum core replies and a short quiet period, so optional review replies do not make the page wait for old fixed counts.
- Automated tests and build pass after the selective participation optimization.
- Faster first-round AI replies have been implemented:
  - all speaking first-round AI role calls now run concurrently instead of one after another.
  - AI messages are still saved in the selected role order after model calls finish.
  - token usage recording and AI review still run after first-round replies are saved.
- The previous "only two first-round speakers" strategy has been rolled back:
  - every AI role with speaking permission participates in the first round again.
  - frontend polling now waits for the current speaking-role count before stopping.
  - AI review remains capped to at most one follow-up reply.
- The chat status text has been simplified:
  - the bottom status now shows `AI 正在回复...`.
  - it no longer shows `AI 正在回复，随后进行互评...`.
- Chat page static assets now include version parameters:
  - `/static/chat.js?v=20260503a`
  - `/static/theme.js?v=20260503a`
  - `/static/app.css?v=20260503a`
  - this forces browsers to load the updated script instead of a cached old `chat.js`.
- AI review trigger has been simplified:
  - if AI review is enabled and at least two first-round AI replies succeed, the app appends one AI review reply.
  - short messages no longer skip AI review.
  - this makes the review feature easier to verify in the browser.
- AI review polling has been fixed:
  - when AI review is enabled, the frontend waits for `speaking role count + 1` AI messages before stopping.
  - this prevents the page from stopping after only the normal first-round replies while the review reply is still being generated.
  - chat page static asset version is now `20260504a` to force browsers to load the updated polling script.
- Automated tests and build pass after the faster reply/status text optimization.
- Remaining checks require the user's browser, model provider, production host, or remote GitHub Actions.

## Automated Evidence

- `env GOCACHE=/tmp/go-build-ai-chat go test -mod=mod ./...` passed.
- `env GOCACHE=/tmp/go-build-ai-chat go build -mod=mod -buildvcs=false ./cmd/server` passed.
- `env GOCACHE=/tmp/go-build-ai-chat go test -mod=mod ./internal/ai` passed.
- `env GOCACHE=/tmp/go-build-ai-chat go test -mod=mod ./internal/http` passed.
- Running-service HTTP checks verify login, chat creation, JSON AI review toggling, tool execution, cleanup, and logout.
- Service tests verify selected route execution for normal replies and AI review.
- HTTP tests verify route metadata visibility.
- Documentation review verified deployment variable/path references against current code and `.env.example`.
- HTTP test verifies `chat_action_error` logging.
- Local CI script verifies test/build quality gate.

## Expected Output

- User confirms the browser acceptance checks.
- User confirms whether AI review replies now feel closer to natural human group-chat follow-up with the real configured model.
- User confirms whether first-round replies feel faster with the real saved model configs.
- User confirms the bottom chat status no longer displays `AI 正在回复，随后进行互评...`.
- User confirms every AI role with speaking permission replies in the first round and the page does not get stuck after two replies.
- User confirms that enabling AI review reliably adds one review reply after the first-round replies.
- User confirms the review reply appears in the current page without manual refresh.
- User confirms whether deployment docs fit the target host.
- User confirms whether remote GitHub Actions runs after pushing.
- Updated `STATUS.md`, `NEXT.md`, `BACKLOG.md`, `DECISIONS.md`, and relevant build/QA/handoff docs.

## Acceptance Method

The next step is accepted when:

- Browser: AI review toggle no longer refreshes the page.
- Browser: file upload appears and AI can use uploaded file text in replies.
- Browser: controlled calculator tool shows success result and failure result.
- Browser: role cards and model settings show route metadata clearly.
- Runtime: real model replies still work with saved model API configs.
- Quality: with AI review enabled, the extra AI replies naturally respond to one previous AI role's point instead of producing a formal supplement/rebuttal report.
- Quality: AI review replies do not mostly repeat the first-round answer, do not summarize every role, and avoid obvious report-style connectors.
- Quality: AI review should be an occasional single follow-up, not two fixed follow-up messages every time.
- Behavior: when AI review is enabled and at least two first-round AI replies succeed, one AI review reply should appear even for short user messages.
- UX: with AI review enabled, polling should wait for all first-round AI replies plus one review reply.
- UX: after sending a message, the bottom status should not mention “随后进行互评”.
- Behavior: every AI role with speaking permission should answer in the first round.
- UX: with three or more speaking roles, the page should not stop polling after only two AI replies.
- Performance: first-round model calls should start concurrently; real perceived speed depends on the slowest selected model/provider response.
- Docs: deployment, observability, and CI docs are acceptable for the user's environment.
- CI: remote GitHub Actions runs after push, if the repo is hosted on GitHub.

## Next After Verification

After user acceptance checks pass, v1.0 can be marked accepted in the user's environment.

## Stop Conditions

Stop before moving on if:

- Any browser main path regresses.
- Real model API replies fail.
- Deployment docs do not fit the user's target environment.
- Remote CI fails for reasons other than runner Go-version availability.

## Ongoing Rule

Every future task round must end by updating:

- `docs/ai/STATUS.md`
- `docs/ai/NEXT.md`
- `docs/ai/BACKLOG.md`
- `docs/ai/DECISIONS.md`
