import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';

import { buildModel, CARD_WIDTH, DEFAULT_RENDER_HEIGHT, EXPORT_SCALE, OUTPUT_WIDTH, renderCard } from './render.mjs';

function baseAgentBlocks() {
  return {
    indicator: {
      expansion: '区间震荡',
      alignment: '信号混杂/分歧',
      noise: '低',
      momentum_detail: '动能不足。',
      conflict_detail: '暂无明显冲突。',
      movement_score: 0.72,
      movement_confidence: 0.78,
    },
    mechanics: {
      leverage_state: '稳定',
      crowding: '多空均衡',
      risk_level: '低',
      open_interest_context: 'OI 平稳。',
      anomaly_detail: '无明显异常。',
      movement_score: 0.7,
      movement_confidence: 0.75,
    },
    structure: {
      regime: '区间震荡',
      last_break: '无法判断',
      quality: '结构清晰',
      pattern: '无法判断',
      volume_action: '量能平稳。',
      candle_reaction: 'K 线反应温和。',
      movement_score: 0.74,
      movement_confidence: 0.8,
    },
  };
}

function createInput(gateOverrides = {}, agentOverrides = {}) {
  return {
    symbol: 'BTCUSDT',
    raw_blocks: {
      gate: {
        decision_action: '观望',
        tradeable: false,
        stop_step: '方向',
        rule_name: 'DIRECTION_MISSING',
        direction_consensus: {
          score: 0.1,
          confidence: 0.2,
          score_threshold: 0.5,
          confidence_threshold: 0.6,
          score_passed: false,
          confidence_passed: false,
        },
        trace: [
          {
            step: '方向',
            ok: false,
            reason: '方向缺失',
          },
        ],
        ...gateOverrides,
      },
      agent: {
        ...baseAgentBlocks(),
        ...agentOverrides,
      },
    },
  };
}

async function renderScenario({ tmpDir, name, input }) {
  const inputPath = path.join(tmpDir, `${name}-input.json`);
  const outputPath = path.join(tmpDir, `${name}-output.png`);
  await fs.writeFile(inputPath, JSON.stringify(input, null, 2));
  const result = await renderCard({ inputPath, outputPath });
  await fs.access(outputPath);
  assert.equal(result.logicalWidth, CARD_WIDTH, `${name} logical width should stay fixed`);
  assert.equal(result.width, OUTPUT_WIDTH, `${name} export width should use hi-res output`);
  assert.ok(result.exportScale >= 1, `${name} export scale should be >= 1`);
  assert.ok(result.height > 0, `${name} render height should be positive`);
  return { inputPath, outputPath, result };
}

async function main() {
  const tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), 'og-card-demo-'));
  const sampleOutputPath = path.join(tmpDir, 'sample-output.png');
  const renderSource = await fs.readFile(new URL('./render.mjs', import.meta.url), 'utf8');

  assert.match(renderSource, /'\.\/brale-icon-only\.png'/, 'card header logo should use local brale-icon-only.png');

  const shortInput = createInput();
  const shortModel = buildModel(shortInput);
  assert.equal(shortModel.sourceCard.sourceLabel, 'Gate 主流程');
  assert.equal(shortModel.sourceCard.lines[0].kind, 'danger');
  assert.match(shortModel.sourceCard.lines[0].text, /停止步骤：方向/);

  const { result } = await renderScenario({ tmpDir, name: 'short', input: shortInput });

  assert.ok(
    result.height < DEFAULT_RENDER_HEIGHT * EXPORT_SCALE,
    `expected cropped height < ${DEFAULT_RENDER_HEIGHT * EXPORT_SCALE}, got ${result.height}`,
  );

  const openSuccessInput = createInput({
    decision_action: '开多',
    tradeable: true,
    stop_step: '',
    rule_name: '',
    trace: [
      { step: '方向', ok: true },
      { step: '清算风险通过', ok: true },
    ],
    direction_consensus: {
      score: 0.76,
      confidence: 0.83,
      score_threshold: 0.5,
      confidence_threshold: 0.6,
      score_passed: true,
      confidence_passed: true,
    },
  });
  const openSuccessModel = buildModel(openSuccessInput);
  assert.equal(openSuccessModel.sourceCard.sourceLabel, 'Gate 总结');
  assert.equal(openSuccessModel.sourceCard.verdictText, '可交易');
  assert.equal(openSuccessModel.sourceCard.lines[0].text, '停止步骤：无（Gate 放行）');
  await renderScenario({ tmpDir, name: 'open-success', input: openSuccessInput });

  const tightenedRiskInput = createInput({
    decision_action: '观望',
    tradeable: false,
    stop_step: '',
    rule_name: 'MECH_RISK',
    action_before: '开多',
    sieve_action: '观望',
    sieve_reason: '清算风险过高',
    trace: [
      { step: '方向', ok: true },
      { step: '清算风险通过', ok: true },
    ],
    direction_consensus: {
      score: 0.71,
      confidence: 0.79,
      score_threshold: 0.5,
      confidence_threshold: 0.6,
      score_passed: true,
      confidence_passed: true,
    },
  }, {
    mechanics: {
      ...baseAgentBlocks().mechanics,
      crowding: '拥挤',
      risk_level: '高',
      anomaly_detail: '拥挤度抬升，风控转为保守。',
    },
  });
  const tightenedRiskModel = buildModel(tightenedRiskInput);
  assert.equal(tightenedRiskModel.sourceCard.sourceLabel, '风控覆写');
  assert.equal(tightenedRiskModel.sourceCard.verdictText, '不可交易');
  assert.equal(tightenedRiskModel.sourceCard.lines[0].text, '停止步骤：Gate 未中断');
  assert.match(tightenedRiskModel.sourceCard.lines[2].text, /风控筛选：开多 → 观望/);
  await renderScenario({ tmpDir, name: 'tightened-risk', input: tightenedRiskInput });

  const tightenSkippedInput = createInput({
    decision_action: '收紧',
    decision_text: '继续持仓（收紧未执行：评分未达标）',
    direction: '多头',
    execution: {
      action: 'tighten',
      evaluated: true,
      executed: false,
      blocked_by: ['评分未达标'],
    },
    trace: [
      { step: '指标', tag: 'tighten', reason: '动能转弱' },
      { step: '结构', tag: 'keep', reason: '结构仍完整' },
    ],
  });
  const tightenSkippedModel = buildModel(tightenSkippedInput);
  assert.equal(tightenSkippedModel.sourceCard.lines[1].text, '持仓处理：收紧未执行 · 原因：评分未达标');
  assert.equal(tightenSkippedModel.sourceCard.lines[2].text, '当前仓位：多头');

  const tightenExecutedInput = createInput({
    decision_action: '收紧',
    decision_text: '执行收紧风控',
    direction: '空头',
    execution: {
      action: 'tighten',
      evaluated: true,
      executed: true,
      stop_loss: 2415.5,
      take_profits: [2399.1, 2377.3],
    },
    trace: [
      { step: '指标', tag: 'tighten', reason: '动能走弱' },
      { step: '市场机制', tag: 'tighten', reason: '拥挤回升' },
    ],
  });
  const tightenExecutedModel = buildModel(tightenExecutedInput);
  assert.equal(tightenExecutedModel.sourceCard.lines[1].text, '持仓处理：已执行收紧');
  assert.equal(tightenExecutedModel.sourceCard.lines[2].text, '当前仓位：空头');
  assert.equal(tightenExecutedModel.sourceCard.lines[3].text, '止损：2415.5 · 止盈：2399.1 / 2377.3');

  const sampleInputPath = path.resolve('./sample-input.json');
  const sampleResult = await renderCard({ inputPath: sampleInputPath, outputPath: sampleOutputPath });
  assert.equal(sampleResult.logicalWidth, CARD_WIDTH, 'sample logical width should stay fixed');
  assert.equal(sampleResult.width, OUTPUT_WIDTH, 'sample export width should use hi-res output');
  assert.ok(sampleResult.height > result.height, 'sample render should be taller than short fixture');
  assert.ok(
    sampleResult.height < (sampleResult.estimatedHeight - 100) * EXPORT_SCALE,
    `expected sample render to stay well below scaled estimate ceiling, got height=${sampleResult.height} estimate=${sampleResult.estimatedHeight} scale=${EXPORT_SCALE}`,
  );
  await fs.access(sampleOutputPath);

  console.log(`ok scale=${EXPORT_SCALE} short=${result.width}x${result.height} open-success tightened-risk sample=${sampleResult.width}x${sampleResult.height}`);

  // ---------- Shutdown card test ----------
  const shutdownInput = {
    card_type: 'shutdown',
    data: {
      reason: '正常停止',
      uptime: '3h25m12s',
    },
  };
  const shutdownModel = buildModel(shutdownInput);
  assert.equal(shutdownModel.symbol, 'BRALE');
  assert.equal(shutdownModel.title, 'Brale 系统停止');
  assert.equal(shutdownModel.sourceCard.verdictText, '🛑 Brale 已停止');
  assert.equal(shutdownModel.sourceCard.lines.length, 2);
  assert.match(shutdownModel.sourceCard.lines[0].text, /停止原因/);
  assert.match(shutdownModel.sourceCard.lines[1].text, /运行时长/);
  await renderScenario({ tmpDir, name: 'shutdown', input: shutdownInput });

  // ---------- Error card test ----------
  const errorInput = {
    card_type: 'error',
    data: {
      severity: 'critical',
      component: 'execution',
      symbol: 'BTCUSDT',
      message: 'Freqtrade API 超时：GET /api/v1/status 连接失败，已重试3次',
    },
  };
  const errorModel = buildModel(errorInput);
  assert.equal(errorModel.symbol, 'BTCUSDT');
  assert.match(errorModel.title, /危急/);
  assert.match(errorModel.sourceCard.lines[0].text, /危急/);
  assert.match(errorModel.sourceCard.lines[1].text, /execution/);
  assert.match(errorModel.sourceCard.lines[2].text, /BTCUSDT/);
  await renderScenario({ tmpDir, name: 'error-critical', input: errorInput });

  // Error card with warn severity and no symbol
  const warnInput = {
    card_type: 'error',
    data: {
      severity: 'warn',
      component: 'market',
      message: 'Fear & Greed API 返回空数据',
    },
  };
  const warnModel = buildModel(warnInput);
  assert.equal(warnModel.symbol, 'BRALE');
  assert.match(warnModel.sourceCard.lines[0].text, /警告/);
  await renderScenario({ tmpDir, name: 'error-warn', input: warnInput });

  console.log('ok shutdown + error card tests passed');

  // ---------- Startup card test ----------
  const startupInput = {
    card_type: 'startup',
    data: {
      symbols: ['BTCUSDT', 'ETHUSDT'],
      intervals: ['15m', '1h'],
      schedule_mode: 'bar',
      bar_interval: '15m',
      balance: 1250.50,
      currency: 'USDT',
      symbol_statuses: [
        { symbol: 'BTCUSDT', intervals: ['15m', '1h'], next_decision: '2025-07-01 12:00', mode: 'live' },
        { symbol: 'ETHUSDT', intervals: ['15m', '1h'], next_decision: '2025-07-01 12:00', mode: 'live' },
      ],
    },
  };
  const startupModel = buildModel(startupInput);
  assert.equal(startupModel.symbol, 'BRALE');
  assert.equal(startupModel.title, 'Brale 系统启动');
  assert.equal(startupModel.sourceCard.sectionTitle, '启动确认');
  assert.equal(startupModel.sourceCard.subtitlePrefix, '状态: ');
  assert.equal(startupModel.metricsLabel, '系统概览');
  assert.equal(startupModel.analysisLabel, '币种详情');
  assert.match(startupModel.sourceCard.verdictText, /Brale 已启动/);
  assert.equal(startupModel.progressCards.length, 4);
  assert.equal(startupModel.analysisItems.length, 2);
  await renderScenario({ tmpDir, name: 'startup', input: startupInput });
  console.log('ok startup card test passed');

  // ---------- Position open card test ----------
  const posOpenInput = {
    card_type: 'position_open',
    symbol: 'ETHUSDT',
    data: {
      direction: '多头',
      entry_price: 3456.78,
      stop_loss: 3400.00,
      take_profits: [3520.00, 3600.00],
      leverage: 5,
      risk_pct: 2.0,
      amount: 0.5,
    },
  };
  const posOpenModel = buildModel(posOpenInput);
  assert.equal(posOpenModel.symbol, 'ETH');
  assert.match(posOpenModel.title, /开仓通知/);
  assert.equal(posOpenModel.sourceCard.sectionTitle, '开仓确认');
  assert.equal(posOpenModel.sourceCard.subtitlePrefix, '操作类型: ');
  assert.equal(posOpenModel.metricsLabel, '开仓概览');
  assert.equal(posOpenModel.analysisLabel, '止盈价位');
  assert.equal(posOpenModel.analysisItems.length, 2);
  await renderScenario({ tmpDir, name: 'position-open', input: posOpenInput });
  console.log('ok position_open card test passed');

  // ---------- Position close card test ----------
  const posCloseInput = {
    card_type: 'position_close',
    symbol: 'BTCUSDT',
    data: {
      direction: '空头',
      entry_price: 68000,
      exit_price: 67200,
      profit: 800,
      profit_ratio: 0.0118,
      exit_reason: '止盈',
      amount: 0.1,
    },
  };
  const posCloseModel = buildModel(posCloseInput);
  assert.match(posCloseModel.title, /平仓/);
  assert.equal(posCloseModel.sourceCard.sectionTitle, '平仓结算');
  assert.equal(posCloseModel.sourceCard.subtitlePrefix, '结算方式: ');
  assert.equal(posCloseModel.metricsLabel, '交易概览');
  assert.equal(posCloseModel.analysisLabel, '');
  assert.equal(posCloseModel.analysisItems.length, 0);
  await renderScenario({ tmpDir, name: 'position-close', input: posCloseInput });
  console.log('ok position_close card test passed');

  // ---------- Risk update card test ----------
  const riskUpdateInput = {
    card_type: 'risk_update',
    symbol: 'BTCUSDT',
    data: {
      direction: '多头',
      entry_price: 67000,
      mark_price: 67800,
      stop_loss: 66500,
      new_stop_loss: 67200,
      take_profits: [68500, 69000],
      leverage: 10,
      source: 'monitor',
      gate_satisfied: true,
    },
  };
  const riskUpdateModel = buildModel(riskUpdateInput);
  assert.match(riskUpdateModel.title, /风控更新/);
  assert.equal(riskUpdateModel.sourceCard.sectionTitle, '风控计划变更');
  assert.equal(riskUpdateModel.sourceCard.subtitlePrefix, '更新来源: ');
  assert.equal(riskUpdateModel.metricsLabel, '风控指标');
  assert.equal(riskUpdateModel.analysisLabel, '止盈价位');
  assert.ok(riskUpdateModel.analysisItems.length > 0);
  await renderScenario({ tmpDir, name: 'risk-update', input: riskUpdateInput });
  console.log('ok risk_update card test passed');

  // ---------- Partial close card test ----------
  const partialCloseInput = {
    card_type: 'partial_close',
    symbol: 'BTCUSDT',
    data: {
      direction: '多头',
      open_rate: 67000,
      close_rate: 68200,
      amount: 0.05,
      realized_profit: 60,
      realized_profit_ratio: 0.0179,
      exit_reason: '止盈1',
      exit_type: '部分平仓',
    },
  };
  const partialCloseModel = buildModel(partialCloseInput);
  assert.match(partialCloseModel.title, /部分平仓/);
  assert.equal(partialCloseModel.sourceCard.sectionTitle, '部分平仓确认');
  assert.equal(partialCloseModel.sourceCard.subtitlePrefix, '操作类型: ');
  assert.equal(partialCloseModel.metricsLabel, '平仓概览');
  assert.equal(partialCloseModel.analysisLabel, '');
  assert.equal(partialCloseModel.analysisItems.length, 0);
  await renderScenario({ tmpDir, name: 'partial-close', input: partialCloseInput });
  console.log('ok partial_close card test passed');

  // ---------- Translation regression tests ----------
  // Since Go now pre-translates all business terms, mapSentence is a pass-through.
  // These tests verify the pass-through behavior works correctly.
  const { mapSentence: mapSentenceFn } = await import('./render.mjs');

  if (typeof mapSentenceFn === 'function') {
    // Pass-through should preserve Chinese text
    const chinese = mapSentenceFn('价格上穿快线EMA');
    assert.equal(chinese, '价格上穿快线EMA', 'mapSentence should pass through Chinese text');

    // Pass-through should handle empty
    const empty = mapSentenceFn('');
    assert.equal(empty, '—', 'mapSentence should return dash for empty');

    // Pass-through should preserve any remaining English (Go handles translation)
    const english = mapSentenceFn('some untranslated text');
    assert.equal(english, 'some untranslated text', 'mapSentence should pass through untranslated text');

    console.log('ok translation-regression: pass-through tests passed');
  } else {
    console.log('warn: mapSentence not exported, skipping translation regression tests');
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
