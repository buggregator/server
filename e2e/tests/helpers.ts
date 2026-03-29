import { Page, expect } from '@playwright/test';

const BUGGREGATOR_URL = 'http://localhost:8000';
const EXAMPLES_URL = 'http://localhost:8080';

/** Trigger an action via the examples Laravel app */
export async function triggerExample(action: string, route = '/example/call'): Promise<void> {
  const res = await fetch(`${EXAMPLES_URL}${route}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: `action=${encodeURIComponent(action)}`,
    redirect: 'follow',
  });
  // Some actions return redirect (302), some return "ok"
  // We don't care about the response, just that it was sent
}

/** Clear all events via the API */
export async function clearEvents(): Promise<void> {
  await fetch(`${BUGGREGATOR_URL}/api/events`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({}),
  });
}

/** Get event count from the API */
export async function getEventCount(type?: string): Promise<number> {
  const url = type
    ? `${BUGGREGATOR_URL}/api/events?type=${type}`
    : `${BUGGREGATOR_URL}/api/events`;
  const res = await fetch(url);
  const json = await res.json();
  return json.data?.length ?? 0;
}

/** Navigate to buggregator and wait for it to load */
export async function openBuggregator(page: Page): Promise<void> {
  await page.goto(BUGGREGATOR_URL);
  await page.waitForLoadState('networkidle');
}

/** Click a sidebar nav item by its visible text */
export async function navigateTo(page: Page, section: string): Promise<void> {
  // The sidebar has links with text like "Sentry", "SMTP", etc.
  await page.locator('nav a, aside a, [class*=sidebar] a, a').filter({ hasText: section }).first().click();
  await page.waitForTimeout(1000);
}

/** Wait for at least N events to appear in the event list */
export async function waitForEvents(page: Page, minCount: number, timeoutMs = 10_000): Promise<void> {
  await expect(async () => {
    const count = await getEventCount();
    expect(count).toBeGreaterThanOrEqual(minCount);
  }).toPass({ timeout: timeoutMs });
}
