// Records what @sentry/node sends: error, message, transaction with spans, logs.
const Sentry = require("@sentry/node");
Sentry.init({
  dsn: "http://publickey@127.0.0.1:18490/42",
  release: "shop@1.2.3", environment: "staging", serverName: "node-01",
  tracesSampleRate: 1.0, enableLogs: true, sendDefaultPii: true,
});
class OrderError extends Error {}
function loadOrder(id) { throw new OrderError(`Order ${id} not found`); }
async function main() {
  Sentry.setUser({ id: "7", email: "dev@example.com" });
  Sentry.setTag("feature", "checkout");
  Sentry.addBreadcrumb({ category: "cart", message: "added item 5", level: "info" });
  try { loadOrder(15); } catch (e) { Sentry.captureException(e); }
  Sentry.captureMessage("Payment gateway slow", "warning");
  await Sentry.startSpan({ name: "GET /orders", op: "http.server" }, async () => {
    await Sentry.startSpan({ name: "SELECT * FROM orders WHERE user_id = ?", op: "db.query", attributes: { "db.system": "postgresql" } }, async () => {
      await new Promise((r) => setTimeout(r, 20));
    });
    try { await fetch("http://127.0.0.1:18490/health"); } catch {}
  });
  Sentry.logger.info(Sentry.logger.fmt`User ${"7"} opened cart`, { cart_items: 3 });
  Sentry.logger.error("Stock sync failed", { sku: "A-1" });
  await Sentry.flush(5000);
}
main();
