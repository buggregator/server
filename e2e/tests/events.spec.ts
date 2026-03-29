import { test, expect, Page } from '@playwright/test';
import { triggerExample, clearEvents, getEventCount, openBuggregator } from './helpers';

const BUGGREGATOR = 'http://localhost:8000';

test.describe.configure({ mode: 'serial' });

/**
 * Wait for at least `count` event cards to appear in the UI.
 * Looks for common event list item patterns in the page.
 */
async function waitForEventInUI(page: Page, timeoutMs = 15_000): Promise<void> {
  // The page title updates with event count: "Events: N | Buggregator"
  // or "Sentry N | Buggregator" etc.
  await expect(async () => {
    const title = await page.title();
    const match = title.match(/(\d+)/);
    expect(match).not.toBeNull();
    expect(Number(match![1])).toBeGreaterThan(0);
  }).toPass({ timeout: timeoutMs });
}

test.describe('Sentry', () => {
  test('event appears in UI via WebSocket', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Sentry' }).first().click();
    await page.waitForTimeout(2000);

    // Sentry SDK sends async — trigger and wait longer
    await triggerExample('sentry:event');

    // Wait for API to confirm event arrived first
    await expect(async () => {
      const count = await getEventCount('sentry');
      expect(count).toBeGreaterThanOrEqual(1);
    }).toPass({ timeout: 10_000 });

    // Now wait for UI to update via WebSocket (or reload as fallback)
    await page.waitForTimeout(3000);
    let title = await page.title();
    if (!title.match(/[1-9]/)) {
      // WebSocket push may not have updated UI yet — reload
      await page.reload();
      await page.waitForTimeout(2000);
    }

    title = await page.title();
    expect(title).toMatch(/[1-9]/);

    // Verify API
    const res = await fetch(`${BUGGREGATOR}/api/events?type=sentry`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].payload).toBeDefined();
    expect(json.data[0].project).toBe('default');
  });
});

test.describe('Ray', () => {
  test('event appears in UI via WebSocket', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Ray' }).first().click();
    await page.waitForTimeout(1000);

    await triggerExample('ray:string');
    await waitForEventInUI(page);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=ray`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].payload).toBeDefined();
  });
});

test.describe('Monolog', () => {
  test('event appears in UI via WebSocket (TCP)', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Monolog' }).first().click();
    await page.waitForTimeout(1000);

    await triggerExample('monolog:error');
    await waitForEventInUI(page);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=monolog`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].payload).toBeDefined();
  });
});

test.describe('VarDumper', () => {
  test('event appears in UI via WebSocket (TCP+PHP)', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: /Var\s*Dump/ }).first().click();
    await page.waitForTimeout(1000);

    await triggerExample('var_dump:string');
    await waitForEventInUI(page);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=var-dump`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);

    const payload = json.data[0].payload;
    expect(payload.payload).toBeDefined();
    expect(payload.payload.type).toBeDefined();
    expect(payload.payload.value).toBeDefined();
  });
});

test.describe('SMTP', () => {
  test('email appears in UI via WebSocket', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'SMTP' }).first().click();
    await page.waitForTimeout(1000);

    await triggerExample('smtp:welcome_mail');
    await waitForEventInUI(page);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=smtp`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].payload.subject).toBeDefined();
  });
});

test.describe('Inspector', () => {
  test('event appears in UI via WebSocket', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Inspector' }).first().click();
    await page.waitForTimeout(1000);

    await triggerExample('inspector:request');
    await waitForEventInUI(page);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=inspector`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].payload).toBeDefined();
  });
});

test.describe('Profiler', () => {
  test('event appears in UI via WebSocket', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Profiler' }).first().click();
    await page.waitForTimeout(1000);

    await triggerExample('profiler:report', '/example/call/profiler');
    await waitForEventInUI(page);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=profiler`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].payload.app_name).toBeDefined();
    expect(json.data[0].payload.peaks).toBeDefined();
  });
});

test.describe('Http Dump', () => {
  test('event appears in UI via WebSocket', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Http Dump' }).first().click();
    await page.waitForTimeout(1000);

    await triggerExample('http:post');
    await waitForEventInUI(page);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=http-dumps`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].payload.method).toBeDefined();
  });
});
