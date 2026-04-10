# 指标替换实施结果

> 日期: 2026-04-10
> 状态: 已实现
> 范围: 引入 `SuperTrend`、引入 `STC`、直接移除 `MACD`
> 依赖: `SuperTrend` 和 `STC` 已本地实现，不再依赖 `github.com/cinar/indicator/v2`

---

## 一、先说结论

这轮已经按目标落完了。

当前代码状态是：

1. `Structure` 层已经新增 `SuperTrend`
2. `Risk` 摘要已经能看到 `SuperTrend`，但它不参与 `nearest_below_entry / nearest_above_entry` 的锚点竞争
3. `Indicator` 层已经新增 `STC`
4. `MACD` 已经从压缩器、配置、默认值、校验、文档和 prototype 里删掉
5. `RSI` 还保留，没动
6. `MFI`、`Donchian`、`VWAP` 这轮还是没做

和最早方案相比，最大的变化只有一个：

**最后没有引入 `cinar/indicator/v2`，而是把 `STC` 和 `SuperTrend` 都做成了本地实现。**

原因很直接：

- `cinar` 是 AGPLv3，不适合继续留在这里
- `STC` 的 channel 计算路径在当前这种有限 slice 输入场景里有阻塞风险
- 这轮真正要的不是“统一第三方指标库”，而是把两条指标链稳稳接进现有系统

所以最后的落地方向是：

- 保留 `go-talib` 继续负责 `EMA / RSI / ATR / BBands`
- `STC` 本地实现
- `SuperTrend` 本地实现
- `required bars` 规则也一起本地化

---

## 二、现在已经是什么样

### 2.1 `Trend` 输出

`TrendCompressedInput` 现在已经带 `supertrend`：

```json
{
  "supertrend": {
    "interval": "1h",
    "state": "UP",
    "level": 3220.45,
    "distance_pct": 0.76
  }
}
```

字段语义已经写死：

- `interval`: 这个 `supertrend` 来自哪个周期
- `state`: `close >= level` 记为 `UP`，否则记为 `DOWN`
- `level`: 当前 trailing level
- `distance_pct`: `abs(close-level) / close * 100`

### 2.2 `Risk` 摘要

`Risk` 摘要顶层现在会带 `supertrend`，但它不会进入 nearest anchor 的候选池。

也就是说：

- `supertrend` 能被风控和 LLM 看见
- `nearest_below_entry / nearest_above_entry` 还是原来那套结构锚点逻辑
- `SuperTrend` 不会去抢 swing、range、order block 的位置

这点和最初要求一致，没有漂。

### 2.3 `Indicator` 输出

`IndicatorCompressedInput` 现在已经没有 `macd`，改成了 `stc`：

```json
{
  "stc": {
    "current": 63.25,
    "last_n": [58.11, 60.42, 61.80, 62.57, 63.25],
    "state": "rising"
  }
}
```

`state` 的规则也已经写死：

- 最近两点差值大于 `2.0`，记 `rising`
- 最近两点差值小于 `-2.0`，记 `falling`
- 其他情况记 `flat`

---

## 三、这轮和原方案的差异

原方案里最开始打算接 `cinar/indicator/v2`，最后实际没有这么做。

现在实际实现和原方案的差异是：

### 3.1 不再新增 `cinar/indicator/v2` 依赖

最终状态是：

- `go.mod` 里没有 `github.com/cinar/indicator/v2`
- `go.sum` 里也没有它

### 3.2 `STC` 改成本地实现

`required bars` 规则和指标序列都已经本地化。

这条链路现在的特点是：

- warmup 规则稳定
- 序列计算可控
- 不依赖上游 channel helper

### 3.3 `SuperTrend` 也改成本地实现

不仅 `SuperTrend` 的数值计算已经本地化，`required bars` 也已经本地化。

这意味着：

- 不再依赖上游 `IdlePeriod()`
- 不再依赖上游 HMA / ATR 的 channel 组合
- 以后如果要调行为，改自己的代码就行

---

## 四、`required bars` 现在怎么定义

这块现在已经统一了，不再是 runtime 一套、validation 一套。

共享 helper 在：

- `internal/config/indicator_requirements.go`

当前规则是：

- `EMARequiredBars(period) = period`
- `RSIRequiredBars(period) = period + 1`
- `ATRRequiredBars(period) = period + 1`
- `STCRequiredBars(fast, slow) = slow + DefaultSTCKPeriod + DefaultSTCDPeriod - 2`
- `SuperTrendRequiredBars(period, multiplier) = period + round(sqrt(period))`

这些规则现在同时服务：

- `requiredKlineLimit()`
- `TrendPresetRequiredBars()`
- `Indicator` runtime
- `Trend` runtime

runtime 现在不再偷偷把周期压小。

也就是说：

- bars 足够，就按配置里的真实 period 算
- bars 不够，就省略这个指标
- 不会出现 validation 允许通过，但 runtime 自动降档的情况

---

## 五、主要代码落点

这轮真正改到的关键文件如下。

### 5.1 配置和 warmup

- `internal/config/indicator_requirements.go`
- `internal/config/trend_presets.go`
- `internal/config/types.go`
- `internal/config/defaults.go`
- `internal/config/validation_symbol.go`
- `internal/runtime/runtime_builder_compression.go`

### 5.2 `SuperTrend`

- `internal/decision/features/trend_supertrend.go`
- `internal/decision/features/trend_compress.go`
- `internal/decision/risk_prompt_anchors.go`

### 5.3 `STC`

- `internal/decision/features/indicator_stc.go`
- `internal/decision/features/indicator_compress.go`

### 5.4 文档、默认配置、prototype

- `configs/symbols/default.toml`
- `configs/symbols/ETHUSDT.toml`
- `docs/configuration.org`
- `webui/config-onboarding-prototype/index.html`
- `webui/config-onboarding-prototype/main.js`
- `internal/config/prompts.go`

---

## 六、测试和验证

这轮已经补了针对性的测试，覆盖了几个关键风险点：

- `STC` 和 `SuperTrend` 在阈值处是否刚好产出
- bars 不够时是否正确省略
- `STC` / `SuperTrend` 是否不会产出 `NaN`
- `SuperTrend` 是否只进 risk summary 顶层，不参与 nearest anchor
- `required bars` 是否在 config 和 runtime 里保持一致
- `STC` / `SuperTrend` 是否有固定回归样本可对拍

当前能通过的验证命令是：

```bash
go test ./internal/config ./internal/decision/features ./internal/decision ./internal/runtime/...
```

全量 `go test ./...` 现在还不是全绿，但剩下的失败点和这轮指标替换无关，主要是仓库里原来就有的基线问题，比如：

- `internal/position`
- `internal/decision/ruleflow`
- `internal/cardimage`
- `internal/onboarding`

所以如果只看这轮指标替换，链路已经跑通。

---

## 七、这轮没做的东西

下面这些之前就说过暂缓，现在状态没变：

- `MFI`
- `Donchian`
- `VWAP`
- mechanics 那几处顺手清理项

这次没有扩 scope。

---

## 八、现在这份文档的定位

这份文档不再是“准备怎么改”的实施方案，而是“已经怎么落了”的结果说明。

如果后面还要继续迭代，建议按这个方向往下走：

1. 维持 `STC` 和 `SuperTrend` 本地实现，不再把第三方依赖接回来
2. 新增指标时，先写清楚它到底服务哪一层
3. 所有 warmup 规则继续走共享 helper
4. 任何会进 risk plan 的字段，先明确它是不是候选锚点

这四点守住，后面就不容易再漂。
