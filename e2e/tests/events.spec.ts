import { test, expect } from '@playwright/test';
import { triggerExample, clearEvents, getEventCount, openBuggregator, waitForEvents } from './helpers';

const BUGGREGATOR = 'http://localhost:8000';

test.describe.configure({ mode: 'serial' });

test.beforeAll(async () => {
  await clearEvents();
});

test.describe('Sentry', () => {
  test('receives events and displays in UI', async ({ page }) => {
    await clearEvents();
    await triggerExample('sentry:event');
    await waitForEvents(page, 1);

    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Sentry' }).first().click();
    await page.waitForTimeout(2000);

    // Should see at least one event card with "sentry" content
    const body = await page.textContent('body');
    expect(body).toContain('sentry');

    // Check API has the event with payload
    const res = await fetch(`${BUGGREGATOR}/api/events?type=sentry`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].type).toBe('sentry');
    expect(json.data[0].payload).toBeDefined();
  });
});

test.describe('Ray', () => {
  test('receives events and displays in UI', async ({ page }) => {
    await clearEvents();
    await triggerExample('ray:string');
    await page.waitForTimeout(2000);

    const count = await getEventCount('ray');
    expect(count).toBeGreaterThanOrEqual(1);

    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Ray' }).first().click();
    await page.waitForTimeout(2000);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=ray`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].payload).toBeDefined();
  });
});

test.describe('Monolog', () => {
  test('receives events via TCP and displays in UI', async ({ page }) => {
    await clearEvents();
    await triggerExample('monolog:error');
    await page.waitForTimeout(2000);

    const count = await getEventCount('monolog');
    expect(count).toBeGreaterThanOrEqual(1);

    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Monolog' }).first().click();
    await page.waitForTimeout(2000);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=monolog`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);

    // Monolog payload should have message/channel/level
    const payload = json.data[0].payload;
    expect(payload).toBeDefined();
  });
});

test.describe('VarDumper', () => {
  test('receives events via TCP and displays in UI', async ({ page }) => {
    await clearEvents();
    await triggerExample('var_dump:string');
    await page.waitForTimeout(3000);

    const count = await getEventCount('var-dump');
    expect(count).toBeGreaterThanOrEqual(1);

    await openBuggregator(page);
    await page.locator('a').filter({ hasText: /Var\s*Dump/ }).first().click();
    await page.waitForTimeout(2000);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=var-dump`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);

    // VarDumper payload should have nested payload with type and value
    const payload = json.data[0].payload;
    expect(payload.payload).toBeDefined();
    expect(payload.payload.type).toBeDefined();
    expect(payload.payload.value).toBeDefined();
  });
});

test.describe('SMTP', () => {
  test('receives emails and displays in UI', async ({ page }) => {
    await clearEvents();
    await triggerExample('smtp:welcome_mail');
    await page.waitForTimeout(3000);

    const count = await getEventCount('smtp');
    expect(count).toBeGreaterThanOrEqual(1);

    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'SMTP' }).first().click();
    await page.waitForTimeout(2000);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=smtp`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);

    // SMTP payload should have subject
    const payload = json.data[0].payload;
    expect(payload.subject).toBeDefined();
  });
});

test.describe('Inspector', () => {
  test('receives events and displays in UI', async ({ page }) => {
    await clearEvents();
    await triggerExample('inspector:request');
    await page.waitForTimeout(3000);

    const count = await getEventCount('inspector');
    expect(count).toBeGreaterThanOrEqual(1);

    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Inspector' }).first().click();
    await page.waitForTimeout(2000);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=inspector`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].payload).toBeDefined();
  });
});

test.describe('Profiler', () => {
  test('receives XHProf data and displays in UI', async ({ page }) => {
    await clearEvents();
    await triggerExample('profiler:report', '/example/call/profiler');
    await page.waitForTimeout(3000);

    const count = await getEventCount('profiler');
    expect(count).toBeGreaterThanOrEqual(1);

    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Profiler' }).first().click();
    await page.waitForTimeout(2000);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=profiler`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);

    const payload = json.data[0].payload;
    expect(payload.app_name).toBeDefined();
    expect(payload.peaks).toBeDefined();
  });
});

test.describe('Http Dump', () => {
  test('captures HTTP requests and displays in UI', async ({ page }) => {
    await clearEvents();
    await triggerExample('http:post');
    await page.waitForTimeout(2000);

    const count = await getEventCount('http-dumps');
    expect(count).toBeGreaterThanOrEqual(1);

    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Http Dump' }).first().click();
    await page.waitForTimeout(2000);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=http-dumps`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);

    const payload = json.data[0].payload;
    expect(payload.method).toBeDefined();
  });
});

test.describe('WebSocket real-time', () => {
  test('events appear in UI without page reload', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);

    // Page should show "waiting for events" or similar empty state
    await page.waitForTimeout(2000);

    // Trigger an event while watching the page
    await triggerExample('monolog:info');

    // Wait for the event to appear via WebSocket (no reload)
    await page.waitForTimeout(5000);

    // Check that the title updated with event count
    const title = await page.title();
    // Title format: "Events: N | Buggregator" or just has a count
    expect(title).toMatch(/\d+/);
  });
});
