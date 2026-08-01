import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {
  DurableEventDedupe,
  isReadOnlyIntent,
  normalizeTextFrame,
  observation,
  safeFallback,
  validateCoreURL,
  validateInbound,
} from '../src/core.mjs';

test('normalizes one WeCom text frame to the shared inbound contract', () => {
  const event = normalizeTextFrame({
    headers: { req_id: 'req-1' },
    body: {
      msgid: 'msg-1',
      aibotid: 'bot-1',
      chatid: 'chat-1',
      chattype: 'group',
      from: { userid: 'user-1' },
      text: { content: '查看 PR #431' },
    },
  }, new Date('2026-07-31T08:00:00Z'));
  assert.equal(event.platform, 'wecom');
  assert.equal(event.event_id, 'msg-1');
  assert.equal(event.text, '查看 PR #431');
  assert.equal(event.received_at, '2026-07-31T08:00:00.000Z');
});

test('policy is fail-closed and observations do not expose raw ids', () => {
  const event = {
    event_id: 'secret-event',
    chat_id: 'secret-chat',
    user_id: 'secret-user',
    text: '查看 PR #431',
  };
  const denied = validateInbound(event, new Set(['other-chat']), new Set());
  assert.deepEqual(denied, { allowed: false, reason: 'chat_not_allowed' });
  const rendered = JSON.stringify(observation('policy', event, denied));
  assert.equal(rendered.includes('secret-event'), false);
  assert.equal(rendered.includes('secret-chat'), false);
  assert.equal(rendered.includes('secret-user'), false);
});

test('write-like commands are rejected before the Review Core', () => {
  const decision = validateInbound({
    event_id: 'event-merge',
    chat_id: 'chat-1',
    user_id: 'user-1',
    text: '合并 PR #431',
  }, new Set(), new Set(), true);
  assert.deepEqual(decision, { allowed: false, reason: 'unsupported_command' });
});

test('empty allowlists fail closed unless allow-all is explicit', () => {
  const event = {
    event_id: 'event-1',
    chat_id: 'chat-1',
    user_id: 'user-1',
    text: '查看 PR #431',
  };
  assert.deepEqual(
    validateInbound(event, new Set(), new Set()),
    { allowed: false, reason: 'allowlist_required' },
  );
  assert.deepEqual(
    validateInbound(event, new Set(), new Set(), true),
    { allowed: true, reason: 'allowed' },
  );
});

test('Review Core endpoint is restricted to loopback HTTP', () => {
  assert.equal(validateCoreURL('http://127.0.0.1:8765/v1/review/inbound'), true);
  assert.equal(validateCoreURL('https://localhost/review'), true);
  assert.equal(validateCoreURL('https://example.com/review'), false);
  assert.equal(validateCoreURL('file:///tmp/review.sock'), false);
});

test('fallback remains GET-only', () => {
  assert.match(safeFallback({ text: '合并 PR #431' }), /GitLink 写入：0/);
});

test('qualified multi-repository commands remain read-only', () => {
  assert.equal(isReadOnlyIntent('查看 Gitlink/gitlink-cli PR #431'), true);
  assert.equal(isReadOnlyIntent('查看 owner/second 待审查'), true);
  assert.equal(isReadOnlyIntent('review queue owner/second'), true);
  assert.equal(isReadOnlyIntent('合并 owner/second PR #42'), false);
});

test('durable dedupe stores only hashed event ids and survives restart', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'wecom-dedupe-'));
  const journal = path.join(directory, 'events.json');
  let now = 100000;
  try {
    const first = new DurableEventDedupe(journal, { ttlMs: 60000, maxEntries: 100, now: () => now });
    assert.equal(first.reserve('raw-message-id'), true);
    assert.equal(first.reserve('raw-message-id'), false);
    const persisted = fs.readFileSync(journal, 'utf8');
    assert.equal(persisted.includes('raw-message-id'), false);

    const restarted = new DurableEventDedupe(journal, { ttlMs: 60000, maxEntries: 100, now: () => now });
    assert.equal(restarted.reserve('raw-message-id'), false);
    assert.equal(restarted.release('raw-message-id'), true);
    assert.equal(restarted.reserve('raw-message-id'), true);
    now += 60001;
    assert.equal(restarted.reserve('raw-message-id'), true);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});
