import { createHash } from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';

export function normalizeTextFrame(frame, now = new Date()) {
  const body = frame?.body ?? {};
  return {
    schema_version: 'gitlink.collab-inbound/v1',
    event_id: String(body.msgid ?? frame?.headers?.req_id ?? ''),
    platform: 'wecom',
    tenant_id: String(body.from?.corpid ?? ''),
    app_id: String(body.aibotid ?? ''),
    chat_id: String(body.chatid ?? body.from?.userid ?? ''),
    user_id: String(body.from?.userid ?? ''),
    conversation: String(body.chattype ?? ''),
    kind: 'message',
    text: String(body.text?.content ?? '').trim(),
    received_at: now.toISOString(),
  };
}

export function validateInbound(event, allowChats, allowUsers, allowAll = false) {
  if (!event.event_id || !event.chat_id || !event.user_id) {
    return { allowed: false, reason: 'invalid_identity' };
  }
  if (!allowAll && allowChats.size === 0 && allowUsers.size === 0) {
    return { allowed: false, reason: 'allowlist_required' };
  }
  if (!allowAll && allowChats.size > 0 && !allowChats.has(event.chat_id)) {
    return { allowed: false, reason: 'chat_not_allowed' };
  }
  if (!allowAll && allowUsers.size > 0 && !allowUsers.has(event.user_id)) {
    return { allowed: false, reason: 'sender_not_allowed' };
  }
  if (!event.text) {
    return { allowed: false, reason: 'empty_text' };
  }
  if (!isReadOnlyIntent(event.text)) {
    return { allowed: false, reason: 'unsupported_command' };
  }
  return { allowed: true, reason: 'allowed' };
}

export function isReadOnlyIntent(text) {
  const repository = String.raw`(?:[^\s/]+\/[^\s/]+)`;
  const supported = new RegExp(
    String.raw`^(?:帮助|help|查看\s*(?:${repository}\s+)?待审查|review\s+queue(?:\s+${repository})?|查看\s+(?:${repository}\s+)?PR\s*#?\d+|刷新\s+(?:${repository}\s+)?PR\s*#?\d+|生成\s+(?:${repository}\s+)?PR\s*#?\d+\s*Review\s*草稿)$`,
    'i',
  );
  return supported.test(String(text ?? '').trim());
}

export function hashIdentifier(value) {
  if (!value) return '';
  return createHash('sha256').update(String(value)).digest('hex').slice(0, 12);
}

export function observation(stage, event, decision = undefined) {
  return {
    schema_version: 'wecom.review-observation/v1',
    stage,
    event_id_hash: hashIdentifier(event?.event_id),
    chat_id_hash: hashIdentifier(event?.chat_id),
    user_id_hash: hashIdentifier(event?.user_id),
    allowed: decision?.allowed,
    reason: decision?.reason,
    observed_at: new Date().toISOString(),
  };
}

export function parseSet(value) {
  return new Set(
    String(value ?? '')
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean),
  );
}

export function parseBoundedInteger(value, fallback, minimum, maximum) {
  const parsed = Number.parseInt(String(value ?? ''), 10);
  if (!Number.isFinite(parsed)) return fallback;
  return Math.max(minimum, Math.min(maximum, parsed));
}

export class DurableEventDedupe {
  constructor(filePath, { ttlMs = 86400000, maxEntries = 10000, now = Date.now } = {}) {
    this.filePath = path.resolve(filePath);
    this.ttlMs = ttlMs;
    this.maxEntries = maxEntries;
    this.now = now;
    this.entries = new Map();
    this.load();
  }

  load() {
    let document;
    try {
      document = JSON.parse(fs.readFileSync(this.filePath, 'utf8'));
    } catch (error) {
      if (error.code === 'ENOENT') return;
      throw new Error(`cannot load WeCom dedupe journal: ${error.message}`);
    }
    if (document?.schema_version !== 'wecom.review-dedupe/v1' || !Array.isArray(document.entries)) {
      throw new Error('invalid WeCom dedupe journal schema');
    }
    const cutoff = this.now() - this.ttlMs;
    for (const entry of document.entries) {
      if (typeof entry?.key === 'string' && Number.isFinite(entry?.seen_at) && entry.seen_at >= cutoff) {
        this.entries.set(entry.key, entry.seen_at);
      }
    }
    this.prune(this.now());
  }

  reserve(eventId) {
    const now = this.now();
    this.prune(now);
    const key = createHash('sha256').update(String(eventId)).digest('hex');
    if (this.entries.has(key)) return false;
    this.entries.set(key, now);
    this.prune(now);
    this.persist();
    return true;
  }

  release(eventId) {
    const key = createHash('sha256').update(String(eventId)).digest('hex');
    if (!this.entries.delete(key)) return false;
    this.persist();
    return true;
  }

  prune(now) {
    const cutoff = now - this.ttlMs;
    for (const [key, seenAt] of this.entries) {
      if (seenAt < cutoff) this.entries.delete(key);
    }
    while (this.entries.size > this.maxEntries) {
      const oldest = [...this.entries.entries()].sort((left, right) => left[1] - right[1])[0];
      if (!oldest) break;
      this.entries.delete(oldest[0]);
    }
  }

  persist() {
    fs.mkdirSync(path.dirname(this.filePath), { recursive: true, mode: 0o700 });
    const temporary = `${this.filePath}.${process.pid}.tmp`;
    const document = {
      schema_version: 'wecom.review-dedupe/v1',
      entries: [...this.entries.entries()].map(([key, seen_at]) => ({ key, seen_at })),
    };
    try {
      fs.writeFileSync(temporary, `${JSON.stringify(document)}\n`, { mode: 0o600 });
      fs.renameSync(temporary, this.filePath);
    } finally {
      try { fs.unlinkSync(temporary); } catch {}
    }
  }
}

export function validateCoreURL(value) {
  if (!String(value ?? '').trim()) return true;
  try {
    const parsed = new URL(value);
    return ['http:', 'https:'].includes(parsed.protocol)
      && ['127.0.0.1', 'localhost', '::1'].includes(parsed.hostname);
  } catch {
    return false;
  }
}

export function safeFallback(event) {
  if (!isReadOnlyIntent(event.text)) {
    return '当前企业微信适配器只开放 GitLink Review 只读查询。支持：帮助、查看 owner/repo 待审查、查看 owner/repo PR #编号、刷新 owner/repo PR #编号、生成 owner/repo PR #编号 Review 草稿。GitLink 写入：0。';
  }
  return '已收到只读 Review 请求，但尚未配置 GITLINK_REVIEW_CORE_URL。GitLink 写入：0。';
}
