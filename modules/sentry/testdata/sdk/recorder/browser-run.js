// Opens the rig page in headless Chromium and runs the SDK scenario.
const { chromium } = require("playwright-core");
(async () => {
  const exe = process.env.HOME + "/.cache/ms-playwright/chromium-1254/chrome-linux64/chrome";
  const b = await chromium.launch({ executablePath: exe });
  const p = await b.newPage();
  p.on("console", (m) => console.log("console:", m.type(), m.text().slice(0, 200)));
  p.on("requestfailed", (r) => console.log("failed:", r.method(), r.url().slice(0, 80), r.failure()?.errorText));
  await p.goto("http://localhost:18490/index.html?page=checkout" + (process.env.DSN ? "&dsn=" + encodeURIComponent(process.env.DSN) : ""));
  console.log(await p.evaluate(() => window.runRig()));
  await p.waitForTimeout(1500);
  await b.close();
})();
