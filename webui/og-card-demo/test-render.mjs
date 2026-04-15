import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';

import { buildModel, CARD_WIDTH, DEFAULT_RENDER_HEIGHT, EXPORT_SCALE, OUTPUT_WIDTH, renderCard } from './render.mjs';

function baseAgentBlocks() {
  return {
    indicator: {
      expansion: 'range',
      alignment: 'mixed',
      noise: 'low',
      momentum_detail: '动能不足。',
      conflict_detail: '暂无明显冲突。',
      movement_score: 0.72,
      movement_confidence: 0.78,
    },
    mechanics: {
      leverage_state: 'stable',
      crowding: 'balanced',
      risk_level: 'low',
      open_interest_context: 'OI 平稳。',
      anomaly_detail: '无明显异常。',
      movement_score: 0.7,
      movement_confidence: 0.75,
    },
    structure: {
      regime: 'range',
      last_break: 'unknown',
      quality: 'clean',
      pattern: 'unknown',
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
        decision_action: 'WAIT',
        tradeable: false,
        stop_step: 'direction',
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
            step: 'direction',
            ok: false,
            reason: 'DIRECTION_MISSING',
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
  assert.match(shortModel.sourceCard.lines[0].text, /停止步骤：清算风险检查|停止步骤：方向/);

  const { result } = await renderScenario({ tmpDir, name: 'short', input: shortInput });

  assert.ok(
    result.height < DEFAULT_RENDER_HEIGHT * EXPORT_SCALE,
    `expected cropped height < ${DEFAULT_RENDER_HEIGHT * EXPORT_SCALE}, got ${result.height}`,
  );

  const openSuccessInput = createInput({
    decision_action: 'OPEN_LONG',
    tradeable: true,
    stop_step: '',
    rule_name: '',
    trace: [
      { step: 'direction', ok: true },
      { step: 'clear_risk', ok: true },
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
    decision_action: 'WAIT',
    tradeable: false,
    stop_step: '',
    rule_name: 'MECH_RISK',
    action_before: 'OPEN_LONG',
    sieve_action: 'WAIT',
    sieve_reason: 'MECH_RISK',
    trace: [
      { step: 'direction', ok: true },
      { step: 'clear_risk', ok: true },
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
      crowding: 'crowded',
      risk_level: 'high',
      anomaly_detail: '拥挤度抬升，风控转为保守。',
    },
  });
  const tightenedRiskModel = buildModel(tightenedRiskInput);
  assert.equal(tightenedRiskModel.sourceCard.sourceLabel, '风控覆写');
  assert.equal(tightenedRiskModel.sourceCard.verdictText, '不可交易');
  assert.equal(tightenedRiskModel.sourceCard.lines[0].text, '停止步骤：Gate 未中断');
  assert.match(tightenedRiskModel.sourceCard.lines[2].text, /风控筛选：(开多|OPEN_LONG) → (观望|等待|WAIT)/);
  await renderScenario({ tmpDir, name: 'tightened-risk', input: tightenedRiskInput });

  const tightenSkippedInput = createInput({
    decision_action: 'TIGHTEN',
    decision_text: '继续持仓（收紧未执行：评分未达标）',
    direction: 'long',
    execution: {
      action: 'tighten',
      evaluated: true,
      executed: false,
      blocked_by: ['score_threshold'],
    },
    trace: [
      { step: 'indicator', tag: 'tighten', reason: '动能转弱' },
      { step: 'structure', tag: 'keep', reason: '结构仍完整' },
    ],
  });
  const tightenSkippedModel = buildModel(tightenSkippedInput);
  assert.equal(tightenSkippedModel.sourceCard.lines[1].text, '持仓处理：收紧未执行 · 原因：评分未达标');
  assert.equal(tightenSkippedModel.sourceCard.lines[2].text, '当前仓位：多头');

  const tightenExecutedInput = createInput({
    decision_action: 'TIGHTEN',
    decision_text: '执行收紧风控',
    direction: 'short',
    execution: {
      action: 'tighten',
      evaluated: true,
      executed: true,
      stop_loss: 2415.5,
      take_profits: [2399.1, 2377.3],
    },
    trace: [
      { step: 'indicator', tag: 'tighten', reason: '动能走弱' },
      { step: 'mechanics', tag: 'tighten', reason: '拥挤回升' },
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

  // ---------- Translation regression tests ----------
  // These verify that mapSentence never produces half-translated event keys.
  const { mapSentence: mapSentenceFn } = await import('./render.mjs');

  const translationTests = [
    {
      in: 'events=price_cross_ema_mid_down',
      bad: ['price_cross_中线EMA_down', 'price_cross_中线EMA'],
      desc: 'event key ema_mid must not be partially translated',
    },
    {
      in: 'events=price_cross_ema_fast_down',
      bad: ['price_cross_快线EMA_down', 'price_cross_快线EMA'],
      desc: 'event key ema_fast must not be partially translated',
    },
    {
      in: 'aroon_strong_bearish signal detected',
      bad: ['aroon_strong_看空'],
      desc: 'aroon event key must not be split by bearish→看空',
    },
    {
      in: 'aroon_strong_bullish signal detected',
      bad: ['aroon_strong_看多'],
      desc: 'aroon event key must not be split by bullish→看多',
    },
    {
      in: 'price_cross_ema_mid_down triggers entry',
      bad: ['price_cross_中线EMA_down'],
      desc: 'inline event key must remain whole',
    },
  ];

  if (typeof mapSentenceFn === 'function') {
    for (const tc of translationTests) {
      const got = mapSentenceFn(tc.in);
      for (const bad of tc.bad) {
        assert.ok(!got.includes(bad), `mapSentence(${JSON.stringify(tc.in)}) produced half-translated output: ${JSON.stringify(got)} (must not contain ${JSON.stringify(bad)}) — ${tc.desc}`);
      }
    }
    console.log(`ok translation-regression: ${translationTests.length} cases passed`);
  } else {
    console.log('warn: mapSentence not exported, skipping translation regression tests');
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
