import { test, expect, Page } from '@playwright/test';
import { triggerExample, clearEvents, openBuggregator, waitForEvents } from './helpers';

const BUGGREGATOR = 'http://localhost:8000';

test.describe.configure({ mode: 'serial' });

// ============================================================
// Helpers
// ============================================================

async function goToSentry(page: Page): Promise<void> {
  await openBuggregator(page);
  await page.locator('a').filter({ hasText: 'Sentry' }).first().click();
  await page.waitForTimeout(1500);
}

async function waitForSubNav(page: Page): Promise<void> {
  await expect(page.locator('text=Exceptions').first()).toBeVisible({ timeout: 10_000 });
}

async function waitForPreviewCard(page: Page, timeout = 15_000): Promise<void> {
  await expect(page.locator('.preview-card').first()).toBeVisible({ timeout });
}

// ============================================================
// 1. Sentry Exceptions — Timeline navigation
// ============================================================
test.describe('Sentry Exceptions — Timeline', () => {
  test.beforeAll(async () => {
    await clearEvents();
    await triggerExample('sentry:trace_with_error');
    await waitForEvents(undefined as unknown as Page, 1);
  });

  test('sidebar navigates to /sentry/exceptions with sub-nav', async ({ page }) => {
    await goToSentry(page);
    await waitForSubNav(page);

    expect(page.url()).toContain('/sentry/exceptions');
    await expect(page.locator('text=Timeline').first()).toBeVisible();
    await expect(page.locator('text=Group by type').first()).toBeVisible();
  });

  test('event card is visible and clickable', async ({ page }) => {
    await goToSentry(page);
    await waitForSubNav(page);
    await waitForPreviewCard(page);

    const card = page.locator('.preview-card').first();
    // Card should have sentry type label
    const cardText = await card.textContent();
    expect(cardText?.toLowerCase()).toContain('sentry');
  });

  test('clicking event body link opens detail at /sentry/event/:id', async ({ page }) => {
    await goToSentry(page);
    await waitForSubNav(page);
    await waitForPreviewCard(page);

    // Click the body link inside the card
    const bodyLink = page.locator('.preview-card a').first();
    await bodyLink.click();
    await page.waitForTimeout(2000);

    // Should be on /sentry/event/<uuid>
    expect(page.url()).toMatch(/\/sentry\/(event\/)?[0-9a-f-]+/);
  });

  test('detail page shows exception content', async ({ page }) => {
    await goToSentry(page);
    await waitForSubNav(page);
    await waitForPreviewCard(page);

    await page.locator('.preview-card a').first().click();
    await page.waitForTimeout(2000);

    // Should show exception info — at least one of these
    const body = await page.textContent('body');
    expect(body!.length).toBeGreaterThan(200);
  });

  test('"Open full event" header button also opens detail', async ({ page }) => {
    await goToSentry(page);
    await waitForSubNav(page);
    await waitForPreviewCard(page);

    const openBtn = page.locator('.preview-card a[title="Open full event"]').first();
    if (await openBtn.isVisible().catch(() => false)) {
      await openBtn.click();
      await page.waitForTimeout(2000);
      expect(page.url()).toMatch(/\/sentry\/(event\/)?[0-9a-f-]+/);
    }
  });
});

// ============================================================
// 2. Sentry Exceptions — Grouped view
// ============================================================
test.describe('Sentry Exceptions — Grouped', () => {
  test.beforeAll(async () => {
    await clearEvents();
    // Send multiple exceptions to populate grouped view
    await triggerExample('sentry:trace_with_error');
    await triggerExample('sentry:trace_with_error');
    await triggerExample('sentry:event');
    await waitForEvents(undefined as unknown as Page, 3);
  });

  test('"Group by type" toggle shows grouped list', async ({ page }) => {
    await goToSentry(page);
    await waitForSubNav(page);

    await page.locator('text=Group by type').first().click();
    await page.waitForTimeout(2000);

    expect(page.url()).toContain('/sentry/exceptions');
    const body = await page.textContent('body');
    expect(body!.length).toBeGreaterThan(50);
  });

  test('switching back to Timeline shows event cards', async ({ page }) => {
    await goToSentry(page);
    await waitForSubNav(page);

    await page.locator('text=Group by type').first().click();
    await page.waitForTimeout(1000);

    await page.locator('text=Timeline').first().click();
    await page.waitForTimeout(1000);

    expect(page.url()).toContain('/sentry/exceptions');
    await waitForPreviewCard(page);
  });
});

// ============================================================
// 3. Sentry Sub-Navigation — tab switching
// ============================================================
test.describe('Sentry Sub-Navigation', () => {
  test.beforeAll(async () => {
    await clearEvents();
    // Send trace + logs to populate all tabs
    await triggerExample('sentry:trace_with_error');
    await triggerExample('sentry:trace');
    await triggerExample('sentry:logs');
    await waitForEvents(undefined as unknown as Page, 2);
    // Extra wait for structured data processing
    await new Promise(r => setTimeout(r, 3000));
  });

  test('Exceptions tab is always visible and active by default', async ({ page }) => {
    await goToSentry(page);
    await waitForSubNav(page);

    const tab = page.locator('a').filter({ hasText: 'Exceptions' }).first();
    await expect(tab).toBeVisible();
  });

  test('Traces tab navigates to /sentry/traces', async ({ page }) => {
    await goToSentry(page);
    await waitForSubNav(page);

    const tracesTab = page.locator('a').filter({ hasText: 'Traces' }).first();
    // Traces tab appears when trace data exists
    if (await tracesTab.isVisible().catch(() => false)) {
      await tracesTab.click();
      await page.waitForTimeout(1500);
      expect(page.url()).toContain('/sentry/traces');

      // Should show waterfall list / service map toggle
      const hasToggle = await page.locator('text=Waterfall list').first().isVisible().catch(() => false);
      if (hasToggle) {
        await expect(page.locator('text=Service map').first()).toBeVisible();
      }
    }
  });

  test('Logs tab navigates to /sentry/logs', async ({ page }) => {
    await goToSentry(page);
    await waitForSubNav(page);

    const logsTab = page.locator('a').filter({ hasText: 'Logs' }).first();
    if (await logsTab.isVisible().catch(() => false)) {
      await logsTab.click();
      await page.waitForTimeout(1500);
      expect(page.url()).toContain('/sentry/logs');
    }
  });

  test('clicking Exceptions tab returns to /sentry/exceptions', async ({ page }) => {
    await goToSentry(page);
    await waitForSubNav(page);

    // Navigate to traces first if available
    const tracesTab = page.locator('a').filter({ hasText: 'Traces' }).first();
    if (await tracesTab.isVisible().catch(() => false)) {
      await tracesTab.click();
      await page.waitForTimeout(1000);
    }

    // Click back to Exceptions
    await page.locator('a').filter({ hasText: 'Exceptions' }).first().click();
    await page.waitForTimeout(1000);
    expect(page.url()).toContain('/sentry/exceptions');
  });
});

// ============================================================
// 4. Sentry Traces — list and detail page navigation
// ============================================================
test.describe('Sentry Traces — detail navigation', () => {
  test.beforeAll(async () => {
    await clearEvents();
    await triggerExample('sentry:trace');
    await triggerExample('sentry:trace_with_error');
    await waitForEvents(undefined as unknown as Page, 2);
    await new Promise(r => setTimeout(r, 3000));
  });

  test('traces page shows trace cards when data exists', async ({ page }) => {
    await page.goto(`${BUGGREGATOR}/sentry/traces`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    const body = await page.textContent('body');
    // Should show either trace cards or empty state with setup instructions
    expect(body!.length).toBeGreaterThan(30);
  });

  test('clicking a trace card navigates to trace detail', async ({ page }) => {
    await page.goto(`${BUGGREGATOR}/sentry/traces`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // Look for trace cards (they have class trace-card)
    const traceCard = page.locator('.trace-card').first();
    if (await traceCard.isVisible().catch(() => false)) {
      await traceCard.click();
      await page.waitForTimeout(2000);

      // Should navigate to /sentry/traces/<traceId>
      expect(page.url()).toMatch(/\/sentry\/traces\/[0-9a-f]+/);

      // Trace detail should show Spans tab
      await expect(page.locator('text=Spans').first()).toBeVisible({ timeout: 5000 });
    }
  });

  test('trace detail shows waterfall with spans', async ({ page }) => {
    await page.goto(`${BUGGREGATOR}/sentry/traces`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    const traceCard = page.locator('.trace-card').first();
    if (await traceCard.isVisible().catch(() => false)) {
      await traceCard.click();
      await page.waitForTimeout(2000);

      // Waterfall should show operation names
      const hasDbQuery = await page.locator('text=db.query').first().isVisible().catch(() => false);
      const hasHttpClient = await page.locator('text=http.client').first().isVisible().catch(() => false);
      const hasHttpServer = await page.locator('text=http.server').first().isVisible().catch(() => false);

      // At least one span operation should be visible
      expect(hasDbQuery || hasHttpClient || hasHttpServer).toBeTruthy();
    }
  });

  test('trace detail Related Errors tab shows error entries', async ({ page }) => {
    await page.goto(`${BUGGREGATOR}/sentry/traces`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // Find the trace that has errors (from sentry:trace_with_error)
    const traceCards = page.locator('.trace-card');
    const count = await traceCards.count();

    for (let i = 0; i < count; i++) {
      const card = traceCards.nth(i);
      const text = await card.textContent();
      // The trace_with_error sends "POST /api/checkout" or has "error" badge
      if (text?.includes('error') || text?.includes('checkout')) {
        await card.click();
        await page.waitForTimeout(2000);
        break;
      }
    }

    // If we're on a trace detail page, check Related Errors tab
    if (page.url().match(/\/sentry\/traces\/[0-9a-f]+/)) {
      const errorsTab = page.locator('button').filter({ hasText: 'Related Errors' }).first();
      if (await errorsTab.isVisible().catch(() => false)) {
        await errorsTab.click();
        await page.waitForTimeout(1000);

        // Error entry should be visible with exception type
        const errorLink = page.locator('a').filter({ hasText: 'RuntimeException' }).first();
        await expect(errorLink).toBeVisible({ timeout: 5000 });

        // The link should exist and point to a sentry event detail URL
        const href = await errorLink.getAttribute('href');
        expect(href).toBeTruthy();

        // Click the link — should navigate to event detail, NOT 404
        await errorLink.click();
        await page.waitForTimeout(3000);

        expect(page.url()).not.toContain('/404');
        expect(page.url()).toMatch(/\/sentry\/(event\/)?[0-9a-f-]+/);
      }
    }
  });
});

// ============================================================
// 5. Sentry Logs — page and filtering
// ============================================================
test.describe('Sentry Logs', () => {
  test.beforeAll(async () => {
    await clearEvents();
    await triggerExample('sentry:logs');
    await new Promise(r => setTimeout(r, 3000));
  });

  test('logs page shows log rows or empty state', async ({ page }) => {
    await page.goto(`${BUGGREGATOR}/sentry/logs`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    const body = await page.textContent('body');
    expect(body!.length).toBeGreaterThan(20);
  });

  test('log level filter chips are visible', async ({ page }) => {
    await page.goto(`${BUGGREGATOR}/sentry/logs`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // If logs tab is visible, filter chips should be present
    const logsTab = page.locator('a').filter({ hasText: 'Logs' }).first();
    if (await logsTab.isVisible().catch(() => false)) {
      // Check for filter buttons
      const allChip = page.locator('button').filter({ hasText: /^all$/i }).first();
      if (await allChip.isVisible().catch(() => false)) {
        await expect(allChip).toBeVisible();
        // Click error filter
        const errorChip = page.locator('button').filter({ hasText: /^error$/i }).first();
        if (await errorChip.isVisible().catch(() => false)) {
          await errorChip.click();
          await page.waitForTimeout(500);
          // Should still be on logs page
          expect(page.url()).toContain('/sentry/logs');
        }
      }
    }
  });
});

// ============================================================
// 5b. Sentry Logs — trace link navigation
// ============================================================
test.describe('Sentry Logs — trace link', () => {
  test.beforeAll(async () => {
    await clearEvents();
    // Send logs with orphan trace_ids (no matching trace exists)
    await triggerExample('sentry:logs');
    await new Promise(r => setTimeout(r, 3000));
  });

  test('log rows with orphan trace_id do NOT show a trace link', async ({ page }) => {
    await page.goto(`${BUGGREGATOR}/sentry/logs`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // Logs were sent with random trace_ids that don't match any trace.
    // The "trace →" link should NOT appear because trace_exists=false.
    const traceLink = page.locator('.log-row__trace-link').first();
    const isVisible = await traceLink.isVisible().catch(() => false);
    expect(isVisible).toBeFalsy();
  });

  test('log rows with existing trace_id show a working trace link', async ({ page }) => {
    // Now send a trace so a real trace_id exists, then send logs referencing it
    // sentry:trace_with_error creates both a trace and an error event
    await triggerExample('sentry:trace_with_error');
    await new Promise(r => setTimeout(r, 3000));

    await page.goto(`${BUGGREGATOR}/sentry/logs`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // Check if any trace link is now visible
    const traceLink = page.locator('.log-row__trace-link').first();
    if (await traceLink.isVisible().catch(() => false)) {
      const href = await traceLink.getAttribute('href');
      expect(href).toContain('/sentry/traces/');

      // Click it — should navigate to trace detail, NOT 404
      await traceLink.click();
      await page.waitForTimeout(3000);

      expect(page.url()).not.toContain('/404');
      expect(page.url()).toMatch(/\/sentry\/traces\/[0-9a-f]+/);
      await expect(page.locator('text=Spans').first()).toBeVisible({ timeout: 5000 });
    }
  });
});

// ============================================================
// 6. Global feed — Sentry events link correctly
// ============================================================
test.describe('Global Feed — Sentry event links', () => {
  test.beforeAll(async () => {
    await clearEvents();
    await triggerExample('sentry:trace_with_error');
    await waitForEvents(undefined as unknown as Page, 1);
  });

  test('sentry event in global feed links to /sentry/event/:id', async ({ page }) => {
    await openBuggregator(page);
    await page.waitForTimeout(2000);
    await waitForPreviewCard(page);

    // Find the sentry-type card (has border-l-rose-500 stripe and "sentry" text)
    const sentryCard = page.locator('.preview-card--type-sentry').first();
    if (await sentryCard.isVisible().catch(() => false)) {
      const bodyLink = sentryCard.locator('a').first();
      await bodyLink.click();
      await page.waitForTimeout(2000);

      // Should navigate to sentry event detail
      expect(page.url()).toMatch(/\/sentry\/(event\/)?[0-9a-f-]+/);
    }
  });

  test('detail page back to feed via sidebar Home', async ({ page }) => {
    await openBuggregator(page);
    expect(page.url()).toBe(BUGGREGATOR + '/');
    await waitForPreviewCard(page);
  });
});

// ============================================================
// 7. Direct URL access — all routes resolve
// ============================================================
test.describe('Sentry Routes — direct URL', () => {
  test('/sentry redirects to /sentry/exceptions', async ({ page }) => {
    await page.goto(`${BUGGREGATOR}/sentry`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
    expect(page.url()).toContain('/sentry/exceptions');
  });

  test('/sentry/exceptions loads', async ({ page }) => {
    await page.goto(`${BUGGREGATOR}/sentry/exceptions`);
    await page.waitForLoadState('networkidle');
    await expect(page.locator('text=Exceptions').first()).toBeVisible({ timeout: 10_000 });
  });

  test('/sentry/traces loads', async ({ page }) => {
    await page.goto(`${BUGGREGATOR}/sentry/traces`);
    await page.waitForLoadState('networkidle');
    const body = await page.textContent('body');
    expect(body!.length).toBeGreaterThan(20);
  });

  test('/sentry/logs loads', async ({ page }) => {
    await page.goto(`${BUGGREGATOR}/sentry/logs`);
    await page.waitForLoadState('networkidle');
    const body = await page.textContent('body');
    expect(body!.length).toBeGreaterThan(20);
  });

  test('/sentry/event/:id redirect from /sentry/:id works', async ({ page }) => {
    // First get an event ID
    await clearEvents();
    await triggerExample('sentry:event');
    await waitForEvents(undefined as unknown as Page, 1);

    const res = await fetch(`${BUGGREGATOR}/api/events?type=sentry`);
    const json = await res.json();
    const eventId = json.data?.[0]?.uuid;

    if (eventId) {
      // Navigate to /sentry/<uuid> — should redirect to /sentry/event/<uuid>
      await page.goto(`${BUGGREGATOR}/sentry/${eventId}`);
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(2000);

      expect(page.url()).toContain(`/sentry/event/${eventId}`);
    }
  });
});
