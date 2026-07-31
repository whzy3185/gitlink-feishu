import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import AiBot, { generateReqId } from '@wecom/aibot-node-sdk';
import {
  normalizeTextFrame,
  observation,
  parseSet,
  safeFallback,
  validateCoreURL,
  validateInbound,
} from './core.mjs';

const botId = String(process.env.WECOM_BOT_ID ?? '').trim();
const secret = String(process.env.WECOM_BOT_SECRET ?? '').trim();
const coreURL = String(process.env.GITLINK_REVIEW_CORE_URL ?? '').trim();
const coreToken = String(process.env.GITLINK_REVIEW_CORE_TOKEN ?? '').trim();
const allowChats = parseSet(process.env.WECOM_ALLOWED_CHAT_IDS);
const allowUsers = parseSet(process.env.WECOM_ALLOWED_USER_IDS);
const allowAll = /^(1|true|yes)$/i.test(String(process.env.WECOM_ALLOW_ALL ?? '').trim());
const lockPath = path.resolve(process.env.WECOM_BRIDGE_LOCK ?? '.local/wecom-review-bridge.lock');

if (!botId || !secret) {
  throw new Error('WECOM_BOT_ID and WECOM_BOT_SECRET are required');
}
if (!validateCoreURL(coreURL)) {
  throw new Error('GITLINK_REVIEW_CORE_URL must use HTTP(S) on a loopback host');
}
if (coreURL && !coreToken) {
  throw new Error('GITLINK_REVIEW_CORE_TOKEN is required when the Review Core is configured');
}
if (!allowAll && allowChats.size === 0 && allowUsers.size === 0) {
  throw new Error('configure a WeCom chat/user allowlist or explicitly set WECOM_ALLOW_ALL=true');
}

fs.mkdirSync(path.dirname(lockPath), { recursive: true, mode: 0o700 });
let lock;
try {
  lock = fs.openSync(lockPath, 'wx', 0o600);
  fs.writeFileSync(lock, JSON.stringify({ pid: process.pid, started_at: new Date().toISOString() }));
} catch (error) {
  if (error.code !== 'EEXIST') throw error;
  let stale = false;
  try {
    const existing = JSON.parse(fs.readFileSync(lockPath, 'utf8'));
    process.kill(Number(existing.pid), 0);
  } catch (lockError) {
    stale = lockError.code === 'ESRCH' || lockError instanceof SyntaxError;
  }
  if (!stale) {
    throw new Error('another WeCom bridge instance is active');
  }
  fs.unlinkSync(lockPath);
  lock = fs.openSync(lockPath, 'wx', 0o600);
  fs.writeFileSync(lock, JSON.stringify({ pid: process.pid, started_at: new Date().toISOString() }));
}

const client = new AiBot.WSClient({
  botId,
  secret,
  maxReconnectAttempts: -1,
  logger: {
    debug: () => {},
    info: () => process.stderr.write('[wecom] sdk info\n'),
    warn: () => process.stderr.write('[wecom][warn] sdk warning\n'),
    error: () => process.stderr.write('[wecom][error] sdk error\n'),
  },
});

client.on('authenticated', () => {
  process.stdout.write(`${JSON.stringify({
    schema_version: 'wecom.review-bridge/v1',
    type: 'ready',
    mode: coreURL ? 'read_only_core' : 'observe_only',
    gitlink_writes: 0,
    observed_at: new Date().toISOString(),
  })}\n`);
});

client.on('message.text', async (frame) => {
  const event = normalizeTextFrame(frame);
  const decision = validateInbound(event, allowChats, allowUsers, allowAll);
  process.stdout.write(`${JSON.stringify(observation('policy', event, decision))}\n`);
  if (!decision.allowed) {
    if (decision.reason === 'unsupported_command') {
      const deniedStreamId = generateReqId('gitlink-review-denied');
      await client.replyStream(frame, deniedStreamId, safeFallback(event), true);
    }
    return;
  }

  const streamId = generateReqId('gitlink-review');
  await client.replyStream(frame, streamId, '正在读取 GitLink Review 上下文…', false);
  let reply = safeFallback(event);
  if (coreURL) {
    try {
      const response = await fetch(coreURL, {
        method: 'POST',
        headers: {
          'content-type': 'application/json',
          authorization: `Bearer ${coreToken}`,
        },
        body: JSON.stringify(event),
        signal: AbortSignal.timeout(60000),
      });
      if (!response.ok) throw new Error(`Review Core HTTP ${response.status}`);
      const result = await response.json();
      reply = String(result.markdown ?? result.message ?? '只读 Review 已完成。GitLink 写入：0。');
    } catch {
      process.stderr.write('[wecom][error] read-only Review Core request failed\n');
      reply = '只读 Review 请求失败，请联系管理员查看脱敏日志。\nGitLink 写入：0。';
    }
  }
  await client.replyStream(frame, streamId, reply, true);
});

function shutdown() {
  client.disconnect();
  try {
    fs.closeSync(lock);
    fs.unlinkSync(lockPath);
  } catch {}
}

process.on('SIGINT', () => {
  shutdown();
  process.exit(0);
});
process.on('SIGTERM', () => {
  shutdown();
  process.exit(0);
});
process.on('exit', shutdown);

client.connect();
