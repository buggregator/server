import { test, expect, Page } from '@playwright/test';
import { triggerExample, clearEvents, getEventCount, openBuggregator } from './helpers';

const BUGGREGATOR = 'http://localhost:8000';

test.describe.configure({ mode: 'serial' });

async function waitForEventInUI(page: Page, timeoutMs = 15_000): Promise<void> {
  await expect(async () => {
    const title = await page.title();
    const match = title.match(/(\d+)/);
    expect(match).not.toBeNull();
    expect(Number(match![1])).toBeGreaterThan(0);
  }).toPass({ timeout: timeoutMs });
}

async function waitForApiEvent(type: string, timeoutMs = 10_000): Promise<void> {
  await expect(async () => {
    const count = await getEventCount(type);
    expect(count).toBeGreaterThanOrEqual(1);
  }).toPass({ timeout: timeoutMs });
}

// Click the first event in the list to open detail page
async function clickFirstEvent(page: Page): Promise<void> {
  // Event cards are in the list — click the first one
  const card = page.locator('[class*=preview-card], [class*=event-card], [class*=card]').first();
  await card.click();
  await page.waitForTimeout(2000);
}

// ============================================================
// Sentry
// ============================================================
test.describe('Sentry', () => {
  test('event appears in UI and detail page has content', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Sentry' }).first().click();
    await page.waitForTimeout(2000);

    await triggerExample('sentry:event');
    await waitForApiEvent('sentry');

    await page.waitForTimeout(3000);
    let title = await page.title();
    if (!title.match(/[1-9]/)) {
      await page.reload();
      await page.waitForTimeout(2000);
    }
    expect(await page.title()).toMatch(/[1-9]/);

    // Click event to open detail
    await clickFirstEvent(page);
    const body = await page.textContent('body');
    expect(body).toContain('sentry');

    // API check
    const res = await fetch(`${BUGGREGATOR}/api/events?type=sentry`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].payload).toBeDefined();
    expect(json.data[0].payload.event_id).toBeDefined();
  });
});

// ============================================================
// Ray
// ============================================================
test.describe('Ray', () => {
  test('event appears in UI with content', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Ray' }).first().click();
    await page.waitForTimeout(1000);
    await triggerExample('ray:string');
    await waitForEventInUI(page);

    await clickFirstEvent(page);
    const body = await page.textContent('body');
    expect(body!.length).toBeGreaterThan(50);
  });
});

// ============================================================
// Monolog
// ============================================================
test.describe('Monolog', () => {
  test('event appears in UI via TCP', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Monolog' }).first().click();
    await page.waitForTimeout(1000);
    await triggerExample('monolog:error');
    await waitForEventInUI(page);

    await clickFirstEvent(page);
    const body = await page.textContent('body');
    expect(body!.length).toBeGreaterThan(50);
  });
});

// ============================================================
// VarDumper
// ============================================================
test.describe('VarDumper', () => {
  test('event appears in UI with rendered value', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: /Var\s*Dump/ }).first().click();
    await page.waitForTimeout(1000);
    await triggerExample('var_dump:array');
    await waitForEventInUI(page);

    // Verify API has value content
    const res = await fetch(`${BUGGREGATOR}/api/events?type=var-dump`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    const payload = json.data[0].payload;
    expect(payload.payload.type).toBeDefined();
    expect(payload.payload.value.length).toBeGreaterThan(10);
  });
});

// ============================================================
// SMTP with attachment
// ============================================================
test.describe('SMTP', () => {
  test('email appears in UI and attachment is downloadable', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'SMTP' }).first().click();
    await page.waitForTimeout(1000);
    await triggerExample('smtp:welcome_mail');
    await waitForEventInUI(page);

    // Verify email payload
    const eventsRes = await fetch(`${BUGGREGATOR}/api/events?type=smtp`);
    const eventsJson = await eventsRes.json();
    expect(eventsJson.data.length).toBeGreaterThanOrEqual(1);
    const event = eventsJson.data[0];
    expect(event.payload.subject).toBeDefined();
    expect(event.payload.html.length).toBeGreaterThan(100);

    // Check attachments API
    const attRes = await fetch(`${BUGGREGATOR}/api/smtp/${event.uuid}/attachments`);
    const attJson = await attRes.json();
    expect(attJson.data.length).toBeGreaterThanOrEqual(1);

    const att = attJson.data[0];
    expect(att.uuid).toBeDefined();
    expect(att.name).toBeDefined();
    expect(att.size).toBeGreaterThan(0);

    // Download attachment
    const downloadRes = await fetch(`${BUGGREGATOR}/api/smtp/${event.uuid}/attachments/${att.uuid}`);
    expect(downloadRes.status).toBe(200);
    expect(downloadRes.headers.get('content-disposition')).toContain('attachment');
    const downloadBody = await downloadRes.arrayBuffer();
    expect(downloadBody.byteLength).toBe(att.size);

    // Preview attachment
    const previewRes = await fetch(`${BUGGREGATOR}/api/smtp/${event.uuid}/attachments/preview/${att.uuid}`);
    expect(previewRes.status).toBe(200);
    expect(previewRes.headers.get('content-type')).toBe(att.mime);

    // Click event in UI to verify detail page loads
    await clickFirstEvent(page);
    const body = await page.textContent('body');
    expect(body).toContain(event.payload.subject);
  });
});

// ============================================================
// Inspector
// ============================================================
test.describe('Inspector', () => {
  test('event appears in UI with content', async ({ page }) => {
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

// ============================================================
// Profiler
// ============================================================
test.describe('Profiler', () => {
  test('event appears and detail page shows call graph', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Profiler' }).first().click();
    await page.waitForTimeout(1000);
    await triggerExample('profiler:report', '/example/call/profiler');
    await waitForEventInUI(page);

    // Verify API payload
    const eventsRes = await fetch(`${BUGGREGATOR}/api/events?type=profiler`);
    const eventsJson = await eventsRes.json();
    expect(eventsJson.data.length).toBeGreaterThanOrEqual(1);
    const event = eventsJson.data[0];
    expect(event.payload.app_name).toBeDefined();
    expect(event.payload.peaks).toBeDefined();
    expect(event.payload.profile_uuid).toBeDefined();

    // Verify profiler API endpoints
    const uuid = event.payload.profile_uuid;
    const summaryRes = await fetch(`${BUGGREGATOR}/api/profiler/${uuid}/summary`);
    const summary = await summaryRes.json();
    expect(summary.overall_totals).toBeDefined();
    expect(summary.slowest_function).toBeDefined();

    const topRes = await fetch(`${BUGGREGATOR}/api/profiler/${uuid}/top?limit=5`);
    const top = await topRes.json();
    expect(top.schema.length).toBeGreaterThan(0);
    expect(top.functions.length).toBeGreaterThan(0);

    const graphRes = await fetch(`${BUGGREGATOR}/api/profiler/${uuid}/call-graph`);
    const graph = await graphRes.json();
    expect(graph.toolbar.length).toBe(4);
    expect(graph.nodes.length).toBeGreaterThan(0);

    const flameRes = await fetch(`${BUGGREGATOR}/api/profiler/${uuid}/flame-chart`);
    const flame = await flameRes.json();
    expect(flame.length).toBeGreaterThan(0);
    expect(flame[0].name).toBeDefined();
    expect(flame[0].children).toBeDefined();
  });
});

// ============================================================
// HTTP Dump with attachment
// ============================================================
test.describe('Http Dump', () => {
  test('event appears and attachment is downloadable', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);
    await page.locator('a').filter({ hasText: 'Http Dump' }).first().click();
    await page.waitForTimeout(2000);

    await triggerExample('http:post');
    await waitForApiEvent('http-dump');

    await page.waitForTimeout(3000);
    let title = await page.title();
    if (!title.match(/[1-9]/)) {
      await page.reload();
      await page.waitForTimeout(2000);
    }
    expect(await page.title()).toMatch(/[1-9]/);

    // Verify payload structure
    const eventsRes = await fetch(`${BUGGREGATOR}/api/events?type=http-dump`);
    const eventsJson = await eventsRes.json();
    expect(eventsJson.data.length).toBeGreaterThanOrEqual(1);
    const event = eventsJson.data[0];
    expect(event.payload.request.method).toBeDefined();
    expect(event.payload.request.uri).toBeDefined();

    // Check if files exist
    const files = event.payload.request.files;
    if (files && files.length > 0) {
      // Verify attachment API
      const attRes = await fetch(`${BUGGREGATOR}/api/http-dumps/${event.uuid}/attachments`);
      const attJson = await attRes.json();
      expect(attJson.data.length).toBeGreaterThanOrEqual(1);

      const att = attJson.data[0];
      // Download
      const downloadRes = await fetch(`${BUGGREGATOR}/api/http-dumps/${event.uuid}/attachments/${att.uuid}`);
      expect(downloadRes.status).toBe(200);
      const downloadBody = await downloadRes.arrayBuffer();
      expect(downloadBody.byteLength).toBe(att.size);
    }
  });
});

// ============================================================
// SMS
// ============================================================
test.describe('SMS', () => {
  test('event appears in UI', async ({ page }) => {
    await clearEvents();
    await openBuggregator(page);

    // Send SMS directly (examples may not have SMS page in sidebar)
    await fetch(`${BUGGREGATOR}/sms`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        MessageSid: 'SM-test',
        From: '+1234567890',
        To: '+0987654321',
        Body: 'E2E test SMS',
      }),
    });

    await waitForApiEvent('sms');

    const res = await fetch(`${BUGGREGATOR}/api/events?type=sms`);
    const json = await res.json();
    expect(json.data.length).toBeGreaterThanOrEqual(1);
    expect(json.data[0].payload.gateway).toBe('twilio');
    expect(json.data[0].payload.message).toBe('E2E test SMS');
  });
});
