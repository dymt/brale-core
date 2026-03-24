import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import React from 'react';
import satori from 'satori';
import { Resvg } from '@resvg/resvg-js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const DEFAULT_INPUT = path.resolve(
  __dirname,
  './sample-input.json',
);
const DEFAULT_OUTPUT = path.resolve(__dirname, 'ethusdt-og-card.png');
const CONFIG_PATH = path.resolve(__dirname, '../../configs/system.toml');

const h = React.createElement;

const valueMap = new Map([
  ['VETO', '否决'],
  ['veto', '否决'],
  ['none', '无方向'],
  ['CONSENSUS_NOT_PASSED', '三路共识未通过'],
  ['contracting', '收敛'],
  ['mixed', '混合/分歧'],
  ['low', '低'],
  ['divergence_reversal', '指标背离(反转风险)'],
  ['neutral', '中性/无明显倾向'],
  ['stable', '稳定'],
  ['long_crowded', '多头拥挤(追高风险)'],
  ['medium', '中'],
  ['否决', '否决'],
]);

const sentenceMap = new Map([
  ['stable', '稳定'],
  ['medium', '中'],
  ['low', '低'],
  ['negative funding', '负资金费率'],
  ['mixed', '混合'],
  ['positive', '正向'],
  ['negative', '负向'],
  ['long crowding', '多头拥挤'],
]);

function mapValue(value) {
  const key = String(value ?? '').trim();
  if (!key) return '';
  return valueMap.get(key) ?? valueMap.get(key.toLowerCase()) ?? key;
}

function mapSentence(text) {
  let output = String(text ?? '').trim();
  if (!output) return '—';
  for (const [source, target] of sentenceMap.entries()) {
    output = output.replaceAll(source, target);
  }
  return output;
}

function emptyDash(value) {
  const text = String(value ?? '').trim();
  return text || '—';
}

function parseNumber(value, fallback = 0) {
  const n = Number(value);
  return Number.isFinite(n) ? n : fallback;
}

function parseBool(value, fallback = false) {
  if (typeof value === 'boolean') return value;
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase();
    if (normalized === 'true') return true;
    if (normalized === 'false') return false;
  }
  return fallback;
}

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

function lerp(min, max, ratio) {
  return min + (max - min) * ratio;
}

function chooseScale(model) {
  const analysisChars = model.analysisLines.reduce((sum, line) => sum + line.length, 0);
  const thresholdLoad = model.thresholdCards.length * 46;
  const titleLoad = (`${model.symbol}${model.reportTimeCN}`).length * 1.4;
  const contentLoad = analysisChars + thresholdLoad + titleLoad;

  const density = clamp((contentLoad - 420) / (980 - 420), 0, 1);
  const inverse = 1 - density;

  return {
    body: Math.round(lerp(21, 31, inverse)),
    section: Math.round(lerp(26, 36, inverse)),
    metric: Math.round(lerp(18, 24, inverse)),
    chip: Math.round(lerp(15, 21, inverse)),
    hero: Math.round(lerp(54, 90, inverse)),
    sub: Math.round(lerp(22, 36, inverse)),
    gap: Math.round(lerp(13, 24, inverse)),
    barHeight: Math.round(lerp(60, 78, inverse)),
    cardMinHeight: Math.round(lerp(132, 186, inverse)),
    cardGap: Math.round(lerp(8, 14, inverse)),
    cardPadY: Math.round(lerp(12, 22, inverse)),
    cardPadX: Math.round(lerp(14, 22, inverse)),
    progressTrackHeight: Math.round(lerp(12, 18, inverse)),
    cardGridGap: Math.round(lerp(10, 16, inverse)),
  };
}

function applyScaleMultiplier(scale, multiplier) {
  return {
    body: clamp(Math.round(scale.body * multiplier), 18, 44),
    section: clamp(Math.round(scale.section * multiplier), 22, 50),
    metric: clamp(Math.round(scale.metric * multiplier), 15, 36),
    chip: clamp(Math.round(scale.chip * multiplier), 13, 32),
    hero: clamp(Math.round(scale.hero * multiplier), 44, 128),
    sub: clamp(Math.round(scale.sub * multiplier), 18, 52),
    gap: clamp(Math.round(scale.gap * multiplier), 10, 36),
    barHeight: clamp(Math.round(scale.barHeight * multiplier), 70, 138),
    cardMinHeight: clamp(Math.round(scale.cardMinHeight * multiplier), 130, 296),
    cardGap: clamp(Math.round(scale.cardGap * multiplier), 6, 18),
    cardPadY: clamp(Math.round(scale.cardPadY * multiplier), 10, 26),
    cardPadX: clamp(Math.round(scale.cardPadX * multiplier), 12, 26),
    progressTrackHeight: clamp(Math.round(scale.progressTrackHeight * multiplier), 10, 22),
    cardGridGap: clamp(Math.round(scale.cardGridGap * multiplier), 8, 20),
  };
}

function estimateLines(text, fontSize, maxWidth, ratio = 0.54) {
  const lineWidth = Math.max(8, Math.floor(maxWidth / Math.max(8, fontSize * ratio)));
  return Math.max(1, Math.ceil(String(text ?? '').length / lineWidth));
}

function estimateChipRows(model, scale, maxWidth) {
  const chips = [
    `ACTION ${model.action}`,
    `REASON ${model.reason}`,
    `DIRECTION ${model.direction}`,
  ];
  let rows = 1;
  let current = 0;
  const gap = Math.max(10, scale.gap - 4);
  for (const chipText of chips) {
    const chipWidth = Math.round(chipText.length * (scale.chip * 0.62) + scale.chip * 2.35);
    if (current > 0 && current + gap + chipWidth > maxWidth) {
      rows += 1;
      current = chipWidth;
    } else {
      current += current > 0 ? gap + chipWidth : chipWidth;
    }
  }
  return rows;
}

function estimateFillRatio(model, scale) {
  const pageHeight = 1898 - 60;
  const bodyWidth = 1484 - 108;

  const topBlock = scale.barHeight + Math.max(18, scale.gap);
  const bottomBlock = Math.max(14, scale.gap - 3);
  const framePadding = 72;
  const dividerAllowance = 8;

  const heroLines = estimateLines(model.symbol, scale.hero, bodyWidth * 0.96, 0.56);
  const timeLines = estimateLines(model.reportTimeCN, Math.max(20, Math.round(scale.hero * 0.64)), bodyWidth * 0.96, 0.56);
  const chipRows = estimateChipRows(model, scale, bodyWidth);

  const heroSection =
    heroLines * scale.hero * 1.08 +
    timeLines * Math.max(20, Math.round(scale.hero * 0.64)) * 1.32 +
    chipRows * (scale.chip * 1.95) +
    Math.max(14, scale.gap - 2) * 2 +
    Math.max(16, scale.gap);

  const analysisTitle = scale.section + 14 + 18;
  const analysisBody = model.analysisLines.reduce(
    (sum, line) => sum + estimateLines(line, scale.body, bodyWidth, 0.56) * scale.body * 1.5,
    0,
  );

  const totalMetricCards = model.summaryCards.length + model.thresholdCards.length;
  const metricRows = Math.ceil(totalMetricCards / 3);
  const thresholdBlock =
    metricRows * scale.cardMinHeight +
    Math.max(0, metricRows - 1) * scale.cardGridGap +
    16;

  const middleSection =
    Math.max(16, scale.gap - 2) +
    analysisTitle +
    analysisBody +
    Math.max(16, scale.gap - 2) +
    thresholdBlock;

  const totalEstimated = topBlock + heroSection + middleSection + bottomBlock + framePadding + dividerAllowance;
  return totalEstimated / pageHeight;
}

function fitScaleToPage(model, baseScale) {
  const minTarget = 0.78;
  const maxTarget = 0.88;
  const target = 0.84;

  let low = 0.8;
  let high = 1.42;
  let bestScale = baseScale;
  let bestDistance = Number.POSITIVE_INFINITY;

  for (let i = 0; i < 11; i += 1) {
    const mid = (low + high) / 2;
    const candidate = applyScaleMultiplier(baseScale, mid);
    const fillRatio = estimateFillRatio(model, candidate);
    const distance = Math.abs(fillRatio - target);
    if (distance < bestDistance && fillRatio <= maxTarget) {
      bestDistance = distance;
      bestScale = candidate;
    }

    if (fillRatio < minTarget) {
      low = mid;
      continue;
    }
    high = mid;
  }

  return bestScale;
}

function buildModel(raw) {
  const gate = raw.raw_blocks.gate;
  const agent = raw.raw_blocks.agent;

  const action = emptyDash(mapValue(gate.decision_action));
  const reason = emptyDash(mapValue(gate.reason_code));
  const direction = emptyDash(mapValue(gate.direction));

  const consensusRaw = gate.direction_consensus ?? {};
  const fallbackConsensusScore = Math.max(agent.indicator.movement_score, agent.mechanics.movement_score, agent.structure.movement_score);
  const fallbackConsensusConfidence = (agent.indicator.movement_confidence + agent.mechanics.movement_confidence + agent.structure.movement_confidence) / 3;
  const consensusScore = parseNumber(consensusRaw.score, fallbackConsensusScore);
  const consensusConfidence = parseNumber(consensusRaw.confidence, fallbackConsensusConfidence);
  const consensusScoreThreshold = parseNumber(consensusRaw.score_threshold, 0.2);
  const consensusConfidenceThreshold = parseNumber(consensusRaw.confidence_threshold, 0.3);

  const scoreRate = consensusScoreThreshold > 0 ? Math.abs(consensusScore) / consensusScoreThreshold : 1;
  const confidenceRate = consensusConfidenceThreshold > 0 ? consensusConfidence / consensusConfidenceThreshold : 1;
  const scorePassed = Object.hasOwn(consensusRaw, 'score_passed')
    ? parseBool(consensusRaw.score_passed, Math.abs(consensusScore) >= consensusScoreThreshold)
    : Math.abs(consensusScore) >= consensusScoreThreshold;
  const confidencePassed = Object.hasOwn(consensusRaw, 'confidence_passed')
    ? parseBool(consensusRaw.confidence_passed, consensusConfidence >= consensusConfidenceThreshold)
    : consensusConfidence >= consensusConfidenceThreshold;

  const summaryCards = [
    {
      title: '共识总分',
      value: consensusScore,
      threshold: consensusScoreThreshold,
      progress: clamp(scoreRate, 0, 1),
      achievedRate: Math.round(scoreRate * 100),
      passed: scorePassed,
      valueText: consensusScore.toFixed(3),
      thresholdText: consensusScoreThreshold.toFixed(3),
    },
    {
      title: '置信度',
      value: consensusConfidence,
      threshold: consensusConfidenceThreshold,
      progress: clamp(confidenceRate, 0, 1),
      achievedRate: Math.round(confidenceRate * 100),
      passed: confidencePassed,
      valueText: consensusConfidence.toFixed(3),
      thresholdText: consensusConfidenceThreshold.toFixed(3),
    },
  ];

  const thresholdCards = [
    {
      title: '指标置信度',
      current: agent.indicator.movement_confidence,
      target: 0.6,
      score: agent.indicator.movement_score,
      color: '#436f84',
    },
    {
      title: '机制置信度',
      current: agent.mechanics.movement_confidence,
      target: 0.6,
      score: agent.mechanics.movement_score,
      color: '#6f7b52',
    },
    {
      title: '结构置信度',
      current: agent.structure.movement_confidence,
      target: 0.6,
      score: agent.structure.movement_score,
      color: '#9a6a4a',
    },
    {
      title: '方向得分',
      current: [agent.indicator.movement_score, agent.mechanics.movement_score, agent.structure.movement_score]
        .reduce((best, score) => (Math.abs(score) > Math.abs(best) ? score : best), 0),
      target: 0.2,
      score: [agent.indicator.movement_score, agent.mechanics.movement_score, agent.structure.movement_score]
        .reduce((best, score) => (Math.abs(score) > Math.abs(best) ? score : best), 0),
      color: '#6c6586',
      useAbsolute: true,
    },
  ].map((item) => ({
    ...item,
    judgeValue: item.useAbsolute ? Math.abs(item.current) : item.current,
    progress: Math.max(0, Math.min(1, (item.useAbsolute ? Math.abs(item.current) : item.current) / item.target)),
    passed: (item.useAbsolute ? Math.abs(item.current) : item.current) >= item.target,
    detail: `当前 ${item.current.toFixed(2)} / 得分 ${item.score.toFixed(2)}`,
  }));

  const failed = thresholdCards.filter((item) => !item.passed).map((item) => `${item.title} < ${item.target.toFixed(2)}`);

  return {
    symbol: raw.symbol,
    action,
    reason,
    direction,
    summaryCards,
    analysisLines: [
      `Indicator | 扩张状态=${mapValue(agent.indicator.expansion)}  一致性=${mapValue(agent.indicator.alignment)}  噪音=${mapValue(agent.indicator.noise)}`,
      `动能细节=${mapSentence(agent.indicator.momentum_detail)}`,
      `冲突细节=${emptyDash(mapSentence(agent.indicator.conflict_detail))}`,
      `Mechanics | 杠杆=${mapValue(agent.mechanics.leverage_state)}  拥挤度=${mapValue(agent.mechanics.crowding)}  风险等级=${mapValue(agent.mechanics.risk_level)}`,
      `持仓量背景=${mapSentence(agent.mechanics.open_interest_context)}`,
      `异常细节=${mapSentence(agent.mechanics.anomaly_detail)}`,
      `Structure | 结构状态=${emptyDash(mapValue(agent.structure.regime))}  最近突破=${emptyDash(mapValue(agent.structure.last_break))}  形态=${emptyDash(mapValue(agent.structure.pattern))}  质量=${emptyDash(mapValue(agent.structure.quality))}`,
    ],
    thresholdCards,
    failed,
    reportTimeCN: '',
  };
}

function toolbar({ center, right, height = 72 }) {
  return h(
    'div',
    {
      style: {
        width: '100%',
        height,
        display: 'flex',
        alignItems: 'center',
        background: '#ffffff',
        borderBottom: '1px solid #e2e8f0',
        padding: '0 40px',
        boxSizing: 'border-box',
      },
    },
    h(
      'div',
      {
        style: {
          display: 'flex',
          alignItems: 'center',
          color: '#334155',
          fontSize: 21,
          fontWeight: 700,
          letterSpacing: '0.08em',
          textTransform: 'uppercase',
        },
      },
      'Brale',
    ),
    h('div', { style: { display: 'flex', marginLeft: 'auto', marginRight: 'auto', fontSize: 18, color: '#64748b', letterSpacing: '0.14em', textTransform: 'uppercase', fontWeight: 600 } }, center),
    h('div', { style: { display: 'flex', fontSize: 20, color: '#334155', fontWeight: 700, letterSpacing: '0.04em' } }, right),
  );
}

function chip(text, tone) {
  return h(
    'div',
    {
      style: {
        display: 'flex',
        padding: '8px 16px',
        background: tone.background,
        color: tone.color,
        borderRadius: 999,
        fontSize: tone.fontSize,
        fontWeight: 700,
        letterSpacing: '0.08em',
        textTransform: 'uppercase',
        border: `1px solid ${tone.border}`,
      },
    },
    text,
  );
}

function section(title, lines, scale, dotColor) {
  return h(
    'div',
    { style: { display: 'flex', flexDirection: 'column', gap: 16 } },
    h(
      'div',
      { style: { display: 'flex', alignItems: 'center', gap: 14 } },
      h('div', { style: { display: 'flex', width: 10, height: 10, borderRadius: 999, background: dotColor } }),
      h('div', { style: { display: 'flex', fontSize: scale.section, fontWeight: 700, color: '#0f172a', letterSpacing: '0.04em', textTransform: 'uppercase' } }, title),
    ),
    h('div', { style: { display: 'flex', width: '100%', borderTop: '1px dashed #cbd5e1' } }),
    ...lines.map((line) =>
      h(
        'div',
        {
          style: {
            display: 'flex',
            alignItems: 'flex-start',
            gap: 12,
            paddingLeft: 2,
          },
        },
        h('div', { style: { display: 'flex', width: 3, minHeight: Math.max(26, scale.body + 2), background: '#cbd5e1', borderRadius: 999 } }),
        h(
          'div',
          {
            style: {
              display: 'flex',
              fontSize: scale.body,
              lineHeight: 1.62,
              color: '#334155',
              whiteSpace: 'pre-wrap',
            },
          },
          line,
        ),
      ),
    ),
  );
}

function consensusSummaryCard(item, scale) {
  const ratio = clamp(item.progress, 0, 1);
  const percent = Math.round(ratio * 100);
  const markerPos = clamp(Math.round(item.threshold * 100), 0, 100);
  return h(
    'div',
    {
      style: {
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'space-between',
        gap: scale.cardGap,
        padding: `${scale.cardPadY}px ${scale.cardPadX}px`,
        background: '#ffffff',
        border: `1px solid ${item.passed ? '#d1fae5' : '#fde7d9'}`,
        borderRadius: 18,
        width: '100%',
        height: '100%',
        flex: 1,
        boxSizing: 'border-box',
        boxShadow: '0 2px 8px rgba(15,23,42,0.04)',
        minWidth: 0,
      },
    },
    h(
      'div',
      { style: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12 } },
      h('div', { style: { display: 'flex', fontSize: scale.metric + 2, color: '#0f172a', fontWeight: 700 } }, item.title),
      h(
        'div',
        {
          style: {
            display: 'flex',
            fontSize: scale.metric - 4,
            color: item.passed ? '#166534' : '#b45309',
            background: item.passed ? '#dcfce7' : '#ffedd5',
            border: `1px solid ${item.passed ? '#86efac' : '#fdba74'}`,
            borderRadius: 999,
            padding: '4px 8px',
            fontWeight: 700,
          },
        },
        item.passed ? '通过' : '未达阈值',
      ),
    ),
    h(
      'div',
      {
        style: {
          display: 'flex',
          fontSize: Math.max(14, scale.metric - 6),
          lineHeight: 1.25,
          color: '#64748b',
          whiteSpace: 'nowrap',
          fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
          fontVariantNumeric: 'tabular-nums',
        },
      },
      `当前 ${item.valueText} / 达成率 ${item.achievedRate}%`,
    ),
    h(
      'div',
      { style: { display: 'flex', position: 'relative', height: scale.progressTrackHeight, borderRadius: 999, background: '#e2e8f0', overflow: 'hidden' } },
      h('div', { style: { display: 'flex', position: 'absolute', left: 0, top: 0, bottom: 0, width: `${(ratio * 100).toFixed(2)}%`, background: item.passed ? '#34d399' : '#fb923c', borderRadius: 999 } }),
      h('div', { style: { display: 'flex', position: 'absolute', left: `${markerPos}%`, top: -1, bottom: -1, width: 2, background: item.passed ? '#0f172a' : '#b91c1c' } }),
    ),
    h(
      'div',
      {
        style: {
          display: 'flex',
          justifyContent: 'space-between',
          color: '#64748b',
          fontSize: scale.metric - 2,
          fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
          fontVariantNumeric: 'tabular-nums',
        },
      },
      h('div', { style: { display: 'flex' } }, `进度 ${percent}%`),
      h('div', { style: { display: 'flex' } }, `阈值 ${item.thresholdText}`),
    ),
  );
}

function thresholdCard(item, scale) {
  const ratio = clamp(item.progress, 0, 1);
  const percent = Math.round(ratio * 100);
  const markerPos = clamp(Math.round(item.target * 100), 0, 100);
  return h(
    'div',
    {
      style: {
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'space-between',
        gap: scale.cardGap,
        padding: `${scale.cardPadY}px ${scale.cardPadX}px`,
        background: '#ffffff',
        border: `1px solid ${item.passed ? '#d1fae5' : '#fde7d9'}`,
        borderRadius: 18,
        width: '100%',
        height: '100%',
        flex: 1,
        boxSizing: 'border-box',
        boxShadow: '0 2px 8px rgba(15,23,42,0.04)',
      },
    },
    h(
      'div',
      { style: { display: 'flex', flexDirection: 'column', gap: Math.max(6, scale.cardGap - 2) } },
    h(
      'div',
      { style: { display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 12 } },
      h('div', { style: { display: 'flex', fontSize: scale.metric + 2, fontWeight: 700, color: '#0f172a' } }, item.title),
      h(
        'div',
        {
          style: {
            display: 'flex',
            fontSize: scale.metric - 4,
            color: item.passed ? '#166534' : '#b45309',
            fontWeight: 700,
            letterSpacing: '0.02em',
            background: item.passed ? '#dcfce7' : '#ffedd5',
            border: `1px solid ${item.passed ? '#86efac' : '#fdba74'}`,
            borderRadius: 999,
            padding: '4px 8px',
          },
        },
        item.passed ? '通过' : '未达阈值',
      ),
    ),
    h(
      'div',
      {
        style: {
          display: 'flex',
          fontSize: scale.metric - 1,
          color: '#64748b',
          fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
          fontVariantNumeric: 'tabular-nums',
        },
      },
      item.detail,
    ),
    ),
    h(
      'div',
      { style: { display: 'flex', flexDirection: 'column', gap: Math.max(6, scale.cardGap - 1) } },
    h(
      'div',
      { style: { display: 'flex', position: 'relative', height: scale.progressTrackHeight, borderRadius: 999, background: '#e2e8f0', overflow: 'hidden' } },
      h('div', { style: { display: 'flex', position: 'absolute', left: 0, top: 0, bottom: 0, width: `${(ratio * 100).toFixed(2)}%`, background: item.passed ? '#34d399' : '#fb923c', borderRadius: 999 } }),
      h('div', { style: { display: 'flex', position: 'absolute', left: `${markerPos}%`, top: -1, bottom: -1, width: 2, background: item.passed ? '#0f172a' : '#b91c1c' } }),
    ),
    h(
      'div',
      { style: { display: 'flex', justifyContent: 'space-between', fontSize: scale.metric - 2, color: '#64748b', fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace', fontVariantNumeric: 'tabular-nums' } },
      h('div', { style: { display: 'flex' } }, `进度 ${percent}%`),
      h('div', { style: { display: 'flex' } }, `阈值 ${item.target.toFixed(2)}`),
    ),
    ),
  );
}

function thresholdSection(model, scale) {
  const cards = [
    ...model.summaryCards.map((item) => ({ kind: 'summary', item })),
    ...model.thresholdCards.map((item) => ({ kind: 'threshold', item })),
  ];
  const cells = cards.length % 3 === 0
    ? cards
    : cards.concat(Array.from({ length: 3 - (cards.length % 3) }, () => null));
  const rows = [cells.slice(0, 3), cells.slice(3, 6)];

  return h(
    'div',
    {
      style: {
        display: 'flex',
        flexDirection: 'column',
        gap: scale.cardGridGap,
      },
    },
    ...rows.map((row) =>
      h(
        'div',
        { style: { display: 'flex', gap: scale.cardGridGap, height: scale.cardMinHeight } },
        ...row.map((cell) =>
          cell
            ? h(
                'div',
                { style: { display: 'flex', flex: 1, minWidth: 0, height: '100%' } },
                cell.kind === 'summary'
                  ? consensusSummaryCard(cell.item, scale)
                  : thresholdCard(cell.item, scale),
              )
            : h('div', { style: { display: 'flex', flex: 1, minWidth: 0, height: '100%' } }),
        ),
      ),
    ),
  );
}

function buildTree(model, meta) {
  const enrichedModel = { ...model, reportTimeCN: meta.reportTimeCN };
  const baseScale = chooseScale(enrichedModel);
  const scale = fitScaleToPage(enrichedModel, baseScale);
  return h(
    'div',
    {
      style: {
        width: '1484px',
        height: '1898px',
        display: 'flex',
        background: '#f8fafc',
        color: '#15212c',
        fontFamily: 'Noto Sans SC',
        padding: '30px',
        boxSizing: 'border-box',
      },
    },
    h(
      'div',
      {
        style: {
          width: '100%',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          background: '#ffffff',
          border: '1px solid #e2e8f0',
          borderRadius: 24,
          boxShadow: '0 2px 10px rgba(15,23,42,0.06)',
          overflow: 'hidden',
        },
      },
      toolbar({
        center: `${meta.runner}`,
        right: 'by:lauk',
        height: scale.barHeight,
      }),
      h(
        'div',
        { style: { display: 'flex', flexDirection: 'column', gap: Math.max(14, scale.gap - 2), padding: '36px 46px', flexGrow: 1 } },
        h(
          'div',
          { style: { display: 'flex', flexDirection: 'column', gap: Math.max(6, scale.gap - 8), maxWidth: '96%' } },
          h('div', { style: { display: 'flex', fontSize: scale.hero, fontWeight: 800, color: '#0f172a', letterSpacing: '-0.03em', lineHeight: 1.05, flexWrap: 'wrap' } }, model.symbol),
          h('div', { style: { display: 'flex', fontSize: Math.max(20, Math.round(scale.hero * 0.64)), fontWeight: 700, color: '#334155', letterSpacing: '-0.01em', lineHeight: 1.15, flexWrap: 'wrap' } }, meta.reportTimeCN),
        ),
        h(
          'div',
          { style: { display: 'flex', gap: Math.max(12, scale.gap - 5), alignItems: 'center', flexWrap: 'wrap' } },
          chip(`ACTION ${model.action}`, { background: '#fee2e2', color: '#7f1d1d', border: '#fecaca', fontSize: scale.chip }),
          chip(`REASON ${model.reason}`, { background: '#f1f5f9', color: '#334155', border: '#e2e8f0', fontSize: scale.chip }),
          chip(`DIRECTION ${model.direction}`, { background: '#ecfeff', color: '#155e75', border: '#bae6fd', fontSize: scale.chip }),
        ),
        h('div', { style: { display: 'flex', borderTop: '1px solid #e2e8f0' } }),
        h(
          'div',
          { style: { display: 'flex', flexDirection: 'column', gap: Math.max(14, scale.gap - 3), flexGrow: 1 } },
          section('Analysis', model.analysisLines, scale, '#94a3b8'),
          h('div', { style: { display: 'flex', borderTop: '1px dashed #cbd5e1' } }),
          thresholdSection(model, scale),
        ),
      ),
    ),
  );
}

async function resolveRunnerAndTime() {
  const config = await fs.readFile(CONFIG_PATH, 'utf8');
  const match = config.match(/^exec_api_key\s*=\s*"(.+)"$/m);
  let runner = 'UNKNOWN';
  if (match) {
    const raw = match[1].trim();
    const envRef = raw.match(/^\$\{(.+)\}$/);
    if (envRef) {
      runner = (process.env[envRef[1]] || envRef[1]).toUpperCase();
    } else {
      runner = raw.toUpperCase();
    }
  }
  const reportTime = new Intl.DateTimeFormat('en-GB', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'Asia/Shanghai',
  }).format(new Date()).replace(',', '');
  const reportTimeCN = new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'Asia/Shanghai',
  }).format(new Date()).replace(/\//g, '-');
  return { runner, reportTime, reportTimeCN };
}

async function loadFonts() {
  const fontsDir = path.resolve(__dirname, 'fonts');
  const [regular, bold] = await Promise.all([
    fs.readFile(path.join(fontsDir, 'NotoSansCJKsc-Regular.otf')),
    fs.readFile(path.join(fontsDir, 'NotoSansCJKsc-Bold.otf')),
  ]);
  return [
    { name: 'Noto Sans SC', data: regular, weight: 400, style: 'normal' },
    { name: 'Noto Sans SC', data: bold, weight: 700, style: 'normal' },
  ];
}

async function main() {
  const inputPath = path.resolve(process.env.OG_INPUT ?? process.argv[2] ?? DEFAULT_INPUT);
  const outputPath = path.resolve(process.env.OG_OUTPUT ?? process.argv[3] ?? DEFAULT_OUTPUT);
  await fs.access(inputPath);
  await fs.access(path.resolve(__dirname, 'fonts/NotoSansCJKsc-Regular.otf'));
  await fs.access(path.resolve(__dirname, 'fonts/NotoSansCJKsc-Bold.otf'));

  const raw = JSON.parse(await fs.readFile(inputPath, 'utf8'));
  const model = buildModel(raw);
  const meta = await resolveRunnerAndTime();
  const fonts = await loadFonts();

  const svg = await satori(buildTree(model, meta), {
    width: 1484,
    height: 1898,
    fonts,
  });

  const resvg = new Resvg(svg, {
    fitTo: { mode: 'width', value: 1484 },
    background: 'white',
  });
  await fs.writeFile(outputPath, resvg.render().asPng());
  console.log(outputPath);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
