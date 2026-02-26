---
name: brale-runtime
description: Brale Runtime 运行时排障与运维执行技能。用于处理 Brale Runtime 的读取检查、受控运维操作、故障调试与错误合同核对（code,msg,request_id,details）。当用户请求运行时状态排查、14 条 runtime API 路由相关操作、生产问题定位或策略确认时触发。
---

# Brale Runtime

按最小权限执行，默认拒绝（deny-by-default），先读后改，再调试。

## 三层工具与使用时机

1. `runtime_read`（只读层）
   - 用于查询状态、读取配置、检查错误返回、核对 request_id 链路。
   - 默认优先使用；如果只读即可回答，不升级到其他层。

2. `runtime_ops`（运维变更层）
   - 用于受控变更，如启停、切换、重试、恢复类操作。
   - 必须先完成 `runtime_read` 证据收集，再执行最小必要变更。

3. `runtime_debug`（深度调试层）
   - 用于深度诊断、现场复现、细粒度调试信息采集。
   - 仅在 capability 已启用且用户给出显式确认后才可执行；否则禁止调用。

## 执行流程

1. 先用 `runtime_read` 明确问题范围和受影响路由。
2. 需要修复时，提出最小化 `runtime_ops` 方案并执行。
3. 仍无法定位时，申请进入 `runtime_debug`，等待显式确认后再继续。
4. 输出结果时固定包含结论、证据、影响面、后续动作。

## 安全与合规

- 不得暴露密钥。
- 不输出 token、密码、私钥、连接串、完整凭证。
- 调试日志若含敏感字段，先脱敏再展示。
- 对未确认动作保持拒绝，不猜测、不越权。

## v1 范围边界

- 明确排除 webhook ingress。
- 明确排除 decision-view UI routes。

## 输出要求

- 关键错误按 `code,msg,request_id,details` 对齐说明。
- 记录使用的层级（`runtime_read` / `runtime_ops` / `runtime_debug`）与升级原因。
- 若未满足调试前置条件，明确写明未执行 `runtime_debug` 的原因。

## 工具清单（v1）

- `runtime_read.get_schedule_status`
- `runtime_read.get_monitor_status`
- `runtime_read.get_position_status`
- `runtime_read.get_position_history`
- `runtime_read.get_decision_latest`
- `runtime_read.get_news_overlay_latest`
- `runtime_read.get_observe_report`
- `runtime_ops.schedule_enable`
- `runtime_ops.schedule_disable`
- `runtime_ops.schedule_symbol_set`
- `runtime_ops.observe_run`
- `runtime_debug.plan_inject`
- `runtime_debug.plan_status`
- `runtime_debug.plan_clear`

## 使用示例

- 只读排查：先用 `runtime_read.get_monitor_status` 与 `runtime_read.get_decision_latest` 获取现状与决策证据。
- 运维修复：在确认需要变更后使用 `runtime_ops.schedule_enable` 或 `runtime_ops.observe_run`，并记录影响范围。
- 调试诊断：仅在 capability 开启且显式确认后，执行 `runtime_debug.plan_status` 或其他 debug 工具；不满足条件则拒绝。
