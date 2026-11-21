<?php
// Records what sentry/sentry (PHP) sends for a cron monitor: in_progress with monitor_config, then ok with duration.
// Runs in Docker (see README): the host is host.docker.internal there; export rewrites it to 127.0.0.1.
require __DIR__ . '/vendor/autoload.php';

\Sentry\init(['dsn' => 'http://publickey@host.docker.internal:18490/42']);

$id = \Sentry\captureCheckIn(
    slug: 'backup-db',
    status: \Sentry\CheckInStatus::inProgress(),
    monitorConfig: new \Sentry\MonitorConfig(
        \Sentry\MonitorSchedule::crontab('*/5 * * * *'),
        checkinMargin: 2,
        maxRuntime: 10,
        timezone: 'UTC',
    ),
);
\Sentry\captureCheckIn(slug: 'backup-db', status: \Sentry\CheckInStatus::ok(), checkInId: $id, duration: 3.2);
