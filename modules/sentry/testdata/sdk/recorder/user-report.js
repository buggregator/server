// The legacy "user_report" envelope, built by @sentry/browser 7's own code
// (captureUserFeedback needs a browser). Prints the body; POST it as
// text/plain;charset=UTF-8 to /api/42/envelope/.
// NODE_PATH=<dir with @sentry/browser@7> node recorder/user-report.js <event_id>
const { createUserFeedbackEnvelope } = require('@sentry/browser/cjs/userfeedback.js');
const { serializeEnvelope } = require('@sentry/utils');
const env = createUserFeedbackEnvelope(
  { event_id: process.argv[2], name: 'Jane Doe', email: 'jane@example.com', comments: 'It broke when I clicked pay' },
  { metadata: { sdk: { name: 'sentry.javascript.browser', version: require('@sentry/browser/package.json').version } } });
process.stdout.write(serializeEnvelope(env));
