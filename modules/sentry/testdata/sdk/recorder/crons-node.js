// Records what @sentry/node sends for cron monitors: withMonitor (crontab) and manual captureCheckIn (interval, error).
const Sentry = require("@sentry/node");
Sentry.init({
  dsn: "http://publickey@127.0.0.1:18490/42",
  release: "shop@1.2.3", serverName: "node-01",
  tracesSampleRate: 0, defaultIntegrations: false, autoSessionTracking: false,
});
async function main() {
  Sentry.withMonitor("nightly-report", () => {
    let s = 0; for (let i = 0; i < 1e6; i++) s += i; return s;
  }, { schedule: { type: "crontab", value: "0 3 * * *" }, checkinMargin: 5, maxRuntime: 30, timezone: "Europe/Berlin" });
  const checkInId = Sentry.captureCheckIn(
    { monitorSlug: "sync-users", status: "in_progress" },
    { schedule: { type: "interval", value: 10, unit: "minute" } },
  );
  Sentry.captureCheckIn({ checkInId, monitorSlug: "sync-users", status: "error", duration: 1.5 });
  await Sentry.flush(5000);
}
main();
