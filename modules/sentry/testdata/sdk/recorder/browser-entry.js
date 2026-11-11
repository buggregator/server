// Records what @sentry/browser sends from a page: error, message, transaction, logs.
import * as Sentry from "@sentry/browser";
Sentry.init({
  dsn: new URLSearchParams(location.search).get("dsn") || "http://publickey@127.0.0.1:18490/42",
  release: "shop@1.2.3", environment: "staging",
  integrations: [Sentry.browserTracingIntegration()],
  tracesSampleRate: 1.0, enableLogs: true, sendDefaultPii: true,
});
class OrderError extends Error {}
function loadOrder(id) { throw new OrderError(`Order ${id} not found`); }
window.runRig = async () => {
  Sentry.setUser({ id: "7", email: "dev@example.com" });
  Sentry.setTag("feature", "checkout");
  Sentry.addBreadcrumb({ category: "cart", message: "added item 5", level: "info" });
  try { loadOrder(15); } catch (e) { Sentry.captureException(e); }
  Sentry.captureMessage("Payment gateway slow", "warning");
  await Sentry.startSpan({ name: "checkout", op: "ui.action" }, async () => {
    await Sentry.startSpan({ name: "render cart", op: "ui.render" }, () => new Promise((r) => setTimeout(r, 20)));
  });
  Sentry.logger.info(Sentry.logger.fmt`User ${"7"} opened cart`, { cart_items: 3 });
  Sentry.logger.error("Stock sync failed", { sku: "A-1" });
  setTimeout(() => { throw new TypeError("cart.items is undefined"); }, 0);
  await new Promise((r) => setTimeout(r, 200));
  await Sentry.flush(5000);
  return "done";
};
