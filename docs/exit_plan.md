# Exit/Entry Plan 接入指南（给 LLM/AI 开发者）

本文面向需要在本项目中新增或调整入场/出场计划的 AI 开发者。目标是：

- 明确应该在哪些代码位置改动。
- 明确每层输出的数据格式约束。
- 用最少改动完成适配，并保持代码风格与现有架构一致。

---

## 1. 先搞清楚：计划是在哪里生成的

核心链路（从配置到执行计划）如下：

1) 策略配置加载与校验
- `internal/config/loader.go`：加载 strategy 配置，应用默认值。
- `internal/config/validation.go`：校验 `risk_management.initial_exit` 与 `entry_mode`。
- `internal/config/initial_exit_validator.go`：把初始出场策略校验委托给 `initexit.ValidatePolicyConfig`。

2) 规则流构建计划
- `internal/decision/ruleflow/engine_payload.go`：把 `risk_management.initial_exit` 注入 ruleflow 输入载荷。
- `internal/decision/ruleflow/node_plan_builder.go`：
  - 入场价由 `resolveEntryPrice(...)` 计算。
  - 出场（止损/止盈）由 `initexit.BuildInitial(...)` 生成。
  - 最终落成 `plan`（map），后续被解析为 `execution.ExecutionPlan`。

3) 初始出场策略引擎
- `internal/risk/initexit/engine.go`：`BuildInitial(...)` 按策略名路由并统一做结果校验。
- `internal/risk/initexit/registry.go`：策略注册中心（`Register/Get/MustGet`）。
- `internal/risk/initexit/types.go`：策略接口 `Policy`、输入 `BuildInput`、输出 `BuildOutput`。
- `internal/risk/initexit/validate.go`：对 `BuildOutput` 做方向/顺序/比例合法化。

4) 计划结果进入执行与风控
- `internal/decision/ruleflow/engine_result.go`：把 `plan` map 解析为 `execution.ExecutionPlan`。
- `internal/execution/types.go`：`ExecutionPlan` 字段契约。
- `internal/position/risk_plan_service.go` + `internal/risk/plan.go`：把计划转为风险计划并持久化。

---

## 2. 最简单适配方式（优先推荐）

如果你只是想调“入场/出场行为”，绝大多数情况下不需要改 Go 代码，只改策略配置即可。

必改文件（按币种）：
- `configs/strategies/ETHUSDT.toml`（或你的目标策略文件）

常用字段：

```toml
[risk_management]
entry_mode = "orderbook" # 可选: orderbook / atr_offset / market

[risk_management.initial_exit]
policy = "atr_structure_v1" # 可选: atr_structure_v1 / fixed_rr_v1 / structure_tp_v1
structure_interval = "auto" # auto 或具体周期(如 "1h")

[risk_management.initial_exit.params]
stop_atr_multiplier = 2.0
stop_min_distance_pct = 0.005
take_profit_rr = [1.5, 3.0]
take_profit_ratios = [0.5, 0.5]
```

说明：
- 这是当前项目里最稳定、最简洁、最“可审计”的方式。
- `policy` 名称大小写不敏感（内部会做标准化）。
- `structure_interval` 若不是 `auto`，必须存在于对应 symbol 的 intervals 中，否则 runtime 构建会失败（见 `internal/runtime/runtime_builder.go` 的校验逻辑）。

---

## 3. 自定义“出场策略”最佳实践（最优雅扩展点）

### 3.1 结论

新增自定义出场策略时，推荐只在 `internal/risk/initexit` 包内扩展一个新 `Policy`，通过注册中心接入。

这是当前代码中最干净、侵入最小、符合现有模式的做法。

### 3.2 最小改动步骤

1) 新增策略文件
- 路径建议：`internal/risk/initexit/<your_policy>_v1.go`

2) 实现 `Policy` 接口
- 必须实现：
  - `Name() string`
  - `ValidateParams(params map[string]any) error`
  - `Build(ctx context.Context, in BuildInput) (BuildOutput, error)`

3) 在 `init()` 中注册
- `Register(yourPolicy{})`
- 不要重复注册同名策略（会 panic）。

4) 在策略 TOML 中切换 policy
- `risk_management.initial_exit.policy = "<your_policy>_v1"`

5) 补测试
- 参考：`internal/risk/initexit/engine_test.go`

### 3.3 推荐骨架代码

```go
package initexit

import (
    "context"
    "fmt"
)

const myPolicyName = "my_policy_v1"

type myPolicy struct{}

func init() {
    Register(myPolicy{})
}

func (myPolicy) Name() string {
    return myPolicyName
}

func (myPolicy) ValidateParams(params map[string]any) error {
    if err := validateBaseStopParams(params); err != nil {
        return err
    }
    // 在这里校验你自己的参数
    // 例如必须存在的数组、范围、长度等
    return nil
}

func (myPolicy) Build(_ context.Context, in BuildInput) (BuildOutput, error) {
    if in.Entry <= 0 {
        return BuildOutput{}, fmt.Errorf("entry is required")
    }

    // 1) 计算 stop
    // 2) 计算 tps
    // 3) 计算分批比例
    stop := /* your stop logic */ 0.0
    tps := /* your tp logic */ []float64{}
    ratios := /* your ratio logic */ []float64{}

    out := BuildOutput{
        StopLoss:         stop,
        TakeProfits:      tps,
        TakeProfitRatios: ratios,
        StopSource:       "my_policy",
        StopReason:       "my_policy",
    }

    return out, nil
}
```

### 3.4 输出必须满足的硬约束

`BuildOutput` 最终会被 `ValidateAndNormalize(...)` 检查（`internal/risk/initexit/validate.go`）：

- `StopLoss > 0`
- long：`stop_loss < entry`
- short：`stop_loss > entry`
- `TakeProfits` 不可为空
- long：`TakeProfits` 必须递增且都大于 `entry`
- short：`TakeProfits` 必须递减且都小于 `entry`
- `TakeProfitRatios`：
  - 若为空会自动均分
  - 若提供则长度必须与 `TakeProfits` 相同
  - 每项 > 0，且总和 > 0（会归一化）

---

## 4. 自定义“入场策略”的方式（仅在必要时）

当前入场逻辑是 `entry_mode` 驱动，不是独立注册表模式。

如果只是调参，优先用现有三种模式：
- `orderbook`
- `atr_offset`
- `market`

若必须新增一个 entry mode（例如 `vwap_pullback`），通常最少改 2 处；其余按需追加：

1) 入场价格实现
- `internal/decision/ruleflow/node_plan_builder.go`
- 在 `resolveEntryPrice(...)` 中新增分支。

2) 配置校验白名单
- `internal/config/validation.go`
- 在 `validateRiskManagementEntry(...)` 扩展允许值。

3) 默认值/文档模板（可选但建议）
- `internal/config/defaults.go`（如果你希望默认策略能用新模式）
- `configs/strategies/default.toml`（给操作者可见的配置模板）

4) 若新模式依赖额外输入（按需）
- `internal/decision/ruleflow/engine_payload.go`
- 例如当前仅 `entry_mode == "orderbook"` 时才注入 orderbook 数据；
  新模式如果依赖类似额外输入，需要同步注入。

建议：
- 入场模式改造要与出场策略解耦，不要把止损止盈逻辑塞进 entry 分支。

---

## 5. 输出格式契约（AI 必须严格遵守）

### 5.1 初始出场策略输出（Policy -> BuildOutput）

类型定义：`internal/risk/initexit/types.go`

```go
type BuildOutput struct {
    StopLoss         float64
    TakeProfits      []float64
    TakeProfitRatios []float64
    StopSource       string
    StopReason       string
}
```

### 5.2 RuleFlow 计划输出（plan map）

构建位置：`internal/decision/ruleflow/node_plan_builder.go`

关键字段（必须兼容）包括：
- `valid`
- `invalid_reason`
- `direction`
- `entry`
- `stop_loss`
- `take_profits`
- `take_profit_ratios`
- `risk_pct`
- `position_size`
- `leverage`
- `position_id`
- `strategy_id`
- `system_config_hash`
- `strategy_config_hash`
- `risk_annotations`（子字段见下）

### 5.3 运行时强类型解析结果（ExecutionPlan）

解析位置：`internal/decision/ruleflow/engine_result.go`

最终落到：`internal/execution/types.go` 的 `ExecutionPlan`。

注意：
- `risk_annotations` 中并非所有 map 字段都会进入强类型结构体。
- 当前强类型主要读取：`stop_source/stop_reason/risk_distance/atr/buffer_atr/max_invest_pct/max_invest_amt/max_leverage/liquidation_price/mmr/fee`。

### 5.4 风控计划持久化格式（RiskPlan）

类型：`internal/risk/plan.go`

核心 JSON 形态：
- `stop_price`
- `tp_levels[]`：`level_id/price/qty_pct/hit`
- `initial_qty`
- `high_water_mark`
- `low_water_mark`

---

## 6. 需要修改哪些配置文件

按“必要/可选”划分：

必要（至少一个）：
- `configs/strategies/<SYMBOL>.toml`
  - 设置 `risk_management.initial_exit.policy`
  - 设置 `risk_management.initial_exit.params`
  - 可选设置 `risk_management.entry_mode`

可选（按你的落地方式）：
- `configs/strategies/default.toml`
  - 让默认策略模板带上你的策略或参数。
- `configs/symbols/<SYMBOL>.toml`
  - 仅当你把 `initial_exit.structure_interval` 设置为固定周期时，确保该周期在 `intervals` 内。
- `internal/config/defaults.go`
  - 修改全局默认值（会影响未显式配置的策略）。

通常不需要改：
- `configs/system.toml`
- `configs/symbols-index.toml`

---

## 7. AI 提交流程（推荐）

1) 先选择策略类型
- 仅调参：只改 TOML。
- 新出场策略：新增 `initexit` policy 文件 + TOML policy 名称。
- 新入场模式：改 `resolveEntryPrice` + 校验白名单 + TOML。

2) 先保证格式契约，再考虑优化
- 优先让 `BuildOutput` 通过 `ValidateAndNormalize`。
- 不要先改执行层或持久化层。

3) 补测试并做最小回归
- 建议至少执行：
  - `go test ./internal/risk/initexit`
  - `go test ./internal/config`
  - `go test ./internal/decision/ruleflow`

4) 常见失败点
- policy 名称拼错或未注册。
- `take_profit_ratios` 长度与 `take_profit_rr`（生成后的 TP 数量）不一致。
- short/long 方向下 TP 顺序反了。
- `structure_interval` 不在 symbol intervals 中。
