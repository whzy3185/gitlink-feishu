import { createHash } from 'node:crypto';

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
  const supported = /^(帮助|help|查看待审查|review queue|查看\s*PR\s*#?\d+|刷新\s*PR\s*#?\d+|生成\s*PR\s*#?\d+\s*Review\s*草稿)$/i;
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
    return '当前企业微信适配器只开放 GitLink Review 只读查询。支持：帮助、查看待审查、查看 PR #编号、刷新 PR #编号、生成 PR #编号 Review 草稿。GitLink 写入：0。';
  }
  return '已收到只读 Review 请求，但尚未配置 GITLINK_REVIEW_CORE_URL。GitLink 写入：0。';
}
