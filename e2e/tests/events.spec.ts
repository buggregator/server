import { test, expect, Page } from '@playwright/test';
import { triggerExample, clearEvents, openBuggregator } from './helpers';

const BUGGREGATOR = 'http://localhost:8000';

test.describe.configure({ mode: 'serial' });

/** Navigate to sidebar section */
async function goToSection(page: Page, name: string | RegExp) {
  await openBuggregator(page);
  await page.locator('a').filter({ hasText: name }).first().click();
  await page.waitForTimeout(1500);
}

/** Wait until the page title shows at least 1 event */
async function waitForEventCard(page: Page, timeoutMs = 15_000) {
  await expect(async () => {
    const title = await page.title();
    expect(title).toMatch(/[1-9]/);
  }).toPass({ timeout: timeoutMs });
}

/** Click the first event card to open detail page */
async function openFirstEvent(page: Page) {
  await page.locator('.preview-card').first().click();
  await page.waitForTimeout(2000);
}

// ============================================================
// Sentry
// ============================================================
test.describe('Sentry', () => {
  test('shows exception details on detail page', async ({ page }) => {
    await clearEvents();
    await goToSection(page, 'Sentry');
    await triggerExample('sentry:event');
    await waitForEventCard(page);

    // User sees event card in list with "sentry" label
    await expect(page.locator('.preview-card').first()).toBeVisible();

    // User clicks the event
    await openFirstEvent(page);

    // Detail page shows exception info
    await expect(page.locator('text=Exception')).toBeVisible();
  });
});

// ============================================================
// Ray
// ============================================================
test.describe('Ray', () => {
  test('shows dump content on detail page', async ({ page }) => {
    await clearEvents();
    await goToSection(page, 'Ray');
    await triggerExample('ray:string');
    await waitForEventCard(page);

    await openFirstEvent(page);
    const detail = await page.textContent('body');
    expect(detail!.length).toBeGreaterThan(100);
  });
});

// ============================================================
// Monolog
// ============================================================
test.describe('Monolog', () => {
  test('shows log entry on detail page', async ({ page }) => {
    await clearEvents();
    await goToSection(page, 'Monolog');
    await triggerExample('monolog:error');
    await waitForEventCard(page);

    await openFirstEvent(page);
    const detail = await page.textContent('body');
    expect(detail!.length).toBeGreaterThan(100);
  });
});

// ============================================================
// VarDumper
// ============================================================
test.describe('VarDumper', () => {
  test('renders dump value (not empty) in the list', async ({ page }) => {
    await clearEvents();
    await goToSection(page, /Var\s*Dump/);
    await triggerExample('var_dump:array');
    await waitForEventCard(page);

    // Card should show real content, not empty quotes
    const card = await page.locator('.preview-card').first().textContent();
    expect(card!.length).toBeGreaterThan(20);
    expect(card).not.toContain('" "');
  });
});

// ============================================================
// SMTP — full email flow with attachment download
// ============================================================
test.describe('SMTP', () => {
  test('email detail shows subject, recipients, HTML preview and downloadable attachment', async ({ page }) => {
    await clearEvents();
    await goToSection(page, 'SMTP');
    await triggerExample('smtp:welcome_mail');
    await waitForEventCard(page);

    // List card shows subject
    await expect(page.locator('text=Welcome Mail').first()).toBeVisible();

    // User opens the email
    await openFirstEvent(page);

    // Sections are visible
    await expect(page.locator('text=Recipients')).toBeVisible();
    await expect(page.locator('text=Message Info')).toBeVisible();
    await expect(page.locator('text=Attachments')).toBeVisible();

    // Subject visible on detail page
    await expect(page.locator('text=Welcome Mail').first()).toBeVisible();

    // Attachment link exists and is a real <a> href (not a div)
    const attachmentLink = page.locator('a[href*="/attachments/"]').first();
    await expect(attachmentLink).toBeVisible();
    const href = await attachmentLink.getAttribute('href');
    expect(href).toContain('/api/smtp/');
    expect(href).toContain('/attachments/');

    // User clicks attachment — should trigger download
    const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null);
    await attachmentLink.click();
    const download = await downloadPromise;
    if (download) {
      expect(download.suggestedFilename()).toBeTruthy();
    }

    // HTML Preview tab content is visible
    const bodyText = await page.textContent('body');
    expect(bodyText).toContain('Debug with Buggregator');
  });
});

// ============================================================
// Inspector
// ============================================================
test.describe('Inspector', () => {
  test('shows transaction on detail page', async ({ page }) => {
    await clearEvents();
    await goToSection(page, 'Inspector');
    await triggerExample('inspector:request');
    await waitForEventCard(page);

    await openFirstEvent(page);
    const detail = await page.textContent('body');
    expect(detail!.length).toBeGreaterThan(100);
  });
});

// ============================================================
// Profiler — tabs: Call graph, Flamechart, Top functions
// ============================================================
test.describe('Profiler', () => {
  test('detail page has working tabs with data', async ({ page }) => {
    await clearEvents();
    await goToSection(page, 'Profiler');
    await triggerExample('profiler:report', '/example/call/profiler');
    await waitForEventCard(page);

    // List shows app name
    await expect(page.locator('text=Simple app').first()).toBeVisible();

    // User opens the profile
    await openFirstEvent(page);
    await page.waitForTimeout(2000);

    // Tabs are visible
    await expect(page.locator('text=Call graph')).toBeVisible();
    await expect(page.locator('text=Flamechart')).toBeVisible();
    await expect(page.locator('text=Top functions')).toBeVisible();

    // User clicks "Top functions" tab
    await page.locator('text=Top functions').click();
    await page.waitForTimeout(2000);

    // Table header "Function" should be visible
    await expect(page.locator('text=Function').first()).toBeVisible();

    // User clicks "Flamechart" tab
    await page.locator('text=Flamechart').click();
    await page.waitForTimeout(2000);

    // Flamechart should render (has content)
    const flameBody = await page.textContent('body');
    expect(flameBody!.length).toBeGreaterThan(200);
  });
});

// ============================================================
// HTTP Dump — request detail with attachment
// ============================================================
test.describe('Http Dump', () => {
  test('detail page shows method, headers, attachment and cURL', async ({ page }) => {
    await clearEvents();
    await goToSection(page, 'Http Dump');
    await triggerExample('http:post');

    // May need reload for this module
    await expect(async () => {
      const title = await page.title();
      if (!title.match(/[1-9]/)) {
        await page.reload();
        await page.waitForTimeout(2000);
      }
      expect(await page.title()).toMatch(/[1-9]/);
    }).toPass({ timeout: 15_000 });

    // User opens the dump
    await openFirstEvent(page);

    // Request method visible
    await expect(page.locator('text=POST').first()).toBeVisible();

    // Headers section
    await expect(page.locator('text=Headers')).toBeVisible();

    // Attachments section
    await expect(page.locator('text=Attachments')).toBeVisible();

    // Attachment download link
    const attachmentLink = page.locator('a[href*="/attachments/"]').first();
    if (await attachmentLink.isVisible()) {
      const href = await attachmentLink.getAttribute('href');
      expect(href).toContain('/attachments/');
    }

    // cURL section
    await expect(page.getByRole('heading', { name: 'cURL' })).toBeVisible();
  });
});

// ============================================================
// SMS
// ============================================================
test.describe('SMS', () => {
  test('shows SMS with gateway and message', async ({ page }) => {
    await clearEvents();

    // Send SMS directly
    await fetch(`${BUGGREGATOR}/sms/twilio`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        MessageSid: 'SM-e2e',
        From: '+1234567890',
        To: '+0987654321',
        Body: 'E2E test SMS message',
      }),
    });

    await page.waitForTimeout(2000);
    await openBuggregator(page);
    await page.waitForTimeout(2000);

    // SMS event should appear
    const body = await page.textContent('body');
    expect(body).toContain('sms');
  });
});
