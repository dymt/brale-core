import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';

import { CARD_WIDTH, DEFAULT_RENDER_HEIGHT, renderCard } from './render.mjs';

async function main() {
  const tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), 'og-card-demo-'));
  const inputPath = path.join(tmpDir, 'short-input.json');
  const outputPath = path.join(tmpDir, 'short-output.png');
  const sampleOutputPath = path.join(tmpDir, 'sample-output.png');

  const shortInput = {
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
      },
      agent: {
        indicator: {
          expansion: 'range',
          alignment: 'mixed',
          noise: 'low',
          momentum_detail: '动能不足。',
          conflict_detail: '暂无明显冲突。',
          movement_score: 0.1,
          movement_confidence: 0.2,
        },
        mechanics: {
          leverage_state: 'stable',
          crowding: 'balanced',
          risk_level: 'low',
          open_interest_context: 'OI 平稳。',
          anomaly_detail: '无明显异常。',
          movement_score: 0.1,
          movement_confidence: 0.2,
        },
        structure: {
          regime: 'range',
          last_break: 'unknown',
          quality: 'clean',
          pattern: 'unknown',
          volume_action: '量能平稳。',
          candle_reaction: 'K 线反应温和。',
          movement_score: 0.1,
          movement_confidence: 0.2,
        },
      },
    },
  };

  await fs.writeFile(inputPath, JSON.stringify(shortInput, null, 2));
  const result = await renderCard({ inputPath, outputPath });

  assert.equal(result.width, CARD_WIDTH, 'render width should stay fixed');
  assert.ok(result.height > 0, 'render height should be positive');
  assert.ok(result.height < DEFAULT_RENDER_HEIGHT, `expected cropped height < ${DEFAULT_RENDER_HEIGHT}, got ${result.height}`);
  await fs.access(outputPath);

  const sampleInputPath = path.resolve('./sample-input.json');
  const sampleResult = await renderCard({ inputPath: sampleInputPath, outputPath: sampleOutputPath });
  assert.equal(sampleResult.width, CARD_WIDTH, 'sample render width should stay fixed');
  assert.ok(sampleResult.height > result.height, 'sample render should be taller than short fixture');
  assert.ok(
    sampleResult.height < sampleResult.estimatedHeight - 100,
    `expected sample render to stay well below estimate ceiling, got height=${sampleResult.height} estimate=${sampleResult.estimatedHeight}`,
  );
  await fs.access(sampleOutputPath);

  console.log(`ok short=${result.width}x${result.height} sample=${sampleResult.width}x${sampleResult.height}`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
