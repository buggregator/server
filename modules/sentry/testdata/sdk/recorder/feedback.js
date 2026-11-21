// User feedback from @sentry/node: an error, feedback on it, feedback alone.
// NODE_PATH=<dir with @sentry/node> node recorder/feedback.js
const Sentry = require('@sentry/node');
Sentry.init({ dsn: 'http://publickey@127.0.0.1:18490/42', defaultIntegrations: false, tracesSampleRate: 0 });
const id = Sentry.captureException(new Error('Checkout failed'));
Sentry.captureFeedback({ message: 'The pay button did nothing', name: 'Jane Doe', email: 'jane@example.com', url: 'https://shop.test/cart', associatedEventId: id });
Sentry.captureFeedback({ message: 'Love the new dark mode', name: 'Bob' });
Sentry.flush(3000);
