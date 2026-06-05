import { test, expect } from '@playwright/test';

const BUGGREGATOR = 'http://localhost:8000';

// Regression for issue #338: the Sentry JS SDK forces a numeric DSN project id,
// which must land in the visible "default" project rather than an orphan project
// named after the id.
test.describe.configure({ mode: 'serial' });

async function sendSentryStore(eventId: string, value: string): Promise<void> {
  const res = await fetch(`${BUGGREGATOR}/api/1/store`, {
    method: 'POST',
    headers: { 'X-Sentry-Auth': 'Sentry sentry_client=sentry.javascript.browser/8.0.0, sentry_key=abc' },
    body: JSON.stringify({
      event_id: eventId,
      platform: 'javascript',
      level: 'error',
      exception: { values: [{ type: 'TypeError', value }] },
    }),
  });
  expect(res.status).toBe(200);
}

async function sendSentryEnvelope(eventId: string, value: string): Promise<void> {
  const body = [
    JSON.stringify({ event_id: eventId }),
    JSON.stringify({ type: 'event' }),
    JSON.stringify({ event_id: eventId, platform: 'javascript', level: 'error', exception: { values: [{ type: 'ReferenceError', value }] } }),
  ].join('\n');
  const res = await fetch(`${BUGGREGATOR}/api/2/envelope/`, {
    method: 'POST',
    headers: { 'X-Sentry-Auth': 'Sentry sentry_key=abc' },
    body,
  });
  expect(res.status).toBe(200);
}

test.beforeAll(async () => {
  await fetch(`${BUGGREGATOR}/api/events`, { method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: '{}' }).catch(() => {});
  await sendSentryStore('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'Issue 338 numeric DSN error (store)');
  await sendSentryEnvelope('bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb', 'Issue 338 numeric DSN error (envelope)');
});

test('numeric-DSN Sentry events render in the default project feed', async ({ page }) => {
  await page.goto(BUGGREGATOR);
  await page.waitForLoadState('networkidle');

  const card = page.locator('.preview-card').first();
  await expect(card).toBeVisible({ timeout: 15_000 });

  const feedText = (await page.locator('body').textContent()) ?? '';
  expect(feedText.toLowerCase()).toContain('sentry');

  await page.screenshot({ path: 'test-results/issue-338-default-feed.png', fullPage: true });
});

test('no orphan numeric project is created; events are stored under "default"', async () => {
  const projects = await (await fetch(`${BUGGREGATOR}/api/projects`)).json();
  const keys = (projects.data ?? []).map((p: { key: string }) => p.key);
  expect(keys).toContain('default');
  expect(keys).not.toContain('1');
  expect(keys).not.toContain('2');

  const events = await (await fetch(`${BUGGREGATOR}/api/events?project=default`)).json();
  const ids = (events.data ?? []).map((e: { uuid: string }) => e.uuid);
  expect(ids).toContain('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa');
  expect(ids).toContain('bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb');
  for (const e of events.data ?? []) {
    expect(e.project).toBe('default');
  }
});
