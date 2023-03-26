# A server for debugging PHP applications and more.

[![Support me on Patreon](https://img.shields.io/endpoint.svg?url=https%3A%2F%2Fshieldsio-patreon.vercel.app%2Fapi%3Fusername%3Dbutschster%26type%3Dpatrons&style=flat)](https://patreon.com/butschster)
[![Downloads](https://img.shields.io/docker/pulls/butschster/buggregator.svg)](https://hub.docker.com/repository/docker/butschster/buggregator)
[![Twitter](https://img.shields.io/badge/twitter-Follow-blue)](https://twitter.com/buggregator)

![Cover image](https://user-images.githubusercontent.com/773481/208718792-eeae35a6-c5a8-4be4-9474-2b96d222e750.png)

Buggregator is a beautiful, lightweight standalone server built on Spiral Framework, NuxtJs
and [RoadRunner](https://github.com/roadrunner-server/roadrunner) underhood. It helps debugging mostly PHP applications
without extra packages.

It runs without installation on multiple platforms via docker and supports: [Xhprof](#1-compatible-with-xhprof),
[Symfony var-dumper](#2-symfony-vardumper-server), [Monolog](#3-monolog-server), [Sentry](#4-compatible-with-sentry),
[SMTP catcher](#5-fake-smtp-server-for-catching-mail) and [Inspector](#6-compatible-with-inspector).

#### Contents

1. [Features](#features)
    - [Xhprof profiler](#1-xhprof-profiler)
    - [Symfony VarDumper server](#2-symfony-vardumper-server)
    - [Fake SMTP server](#3-fake-smtp-server-for-catching-mail)
    - [Sentry server](#4-compatible-with-sentry-reports)
    - [Monolog server](#5-monolog-server)
    - [Inspector](#6-compatible-with-inspector-reports)
2. [Installation](#installation)
    - [Docker image](#docker-image)
    - [Docker compose](#docker-compose)
3. [Configuration](#configuration)
4. [Contributing](#contributing)
5. [License](#license)

---

## 1. Xhprof profiler

XHProf is a light-weight hierarchical and instrumentation based profiler. During the data collection phase, it keeps
track of call counts and inclusive metrics for arcs in the dynamic callgraph of a program. It computes exclusive metrics
in the reporting/post processing phase, such as wall (elapsed) time, CPU time and memory usage.

![xhprof](https://user-images.githubusercontent.com/773481/208724383-3790a3e1-9ebe-4616-8d4d-d1869f8f2b7c.png)

### Installation

1. Install Xhprof extension
   One of the way to install Xhprof is to use [PECL](https://pecl.php.net/package/xhprof) package.

```bash
pear channel-update pear.php.net
pecl install xhprof
```

2. Install the xhprof package

If you are using Spiral Framework you just need t install
the [spiral/profiler](https://github.com/spiral/profiler/tree/3.0) package.

```bash
composer require --dev spiral/profiler:^3.0
```

### Configuration

```dotenv
PROFILER_ENDPOINT=http://127.0.0.1:8000/api/profiler/store
PROFILER_APP_NAME=My super app
```

> Note: Read more about package usage in the [documentation](https://github.com/spiral/profiler/tree/3.0).

## 2. Symfony [VarDumper server](https://symfony.com/doc/current/components/var_dumper.html#the-dump-server)

The `dump()` and `dd()` functions output its contents in the same browser window or console terminal as your own
application. Sometimes mixing the real output with the debug output can be confusing. That’s why this Buggregator can be
used to collect all the dumped data.

![var-dumper](https://user-images.githubusercontent.com/773481/208727353-b8201775-c360-410b-b5c8-d83843d388ff.png)

### Installation

```bash
composer require --dev symfony/var-dumper
```

### Configuration

You should change dumper format to `server` for var-dumper component. There is a `VAR_DUMPER_FORMAT` env variable in the
package to do it.

```dotenv
VAR_DUMPER_FORMAT=server
VAR_DUMPER_SERVER=127.0.0.1:9912 # Default value
```

via PHP

```php
// Plain PHP
$_SERVER['VAR_DUMPER_FORMAT'] = 'server';
$_SERVER['VAR_DUMPER_SERVER'] = '127.0.0.1:9912';
```

Or you can also define this env var as follows:

```bash
VAR_DUMPER_FORMAT=server php app.php route:list
```

----

## 3. Fake SMTP server for catching mail

Buggregator also is an email testing tool that makes it super easy to install and configure a local email server. It
sets up a fake SMTP server, and you can configure your preferred web applications to use Buggregator’s SMTP server to
send and receive emails. For instance, you can configure a local WordPress site to use it for email deliveries.

![smtp](https://user-images.githubusercontent.com/773481/208727862-229fda5f-3504-4377-921e-03f0ff602cb9.png)

### Configuration

**Env variables**

```dotenv
// Spiral Framework or Symfony
MAILER_DSN=smtp://127.0.0.1:1025

# Laravel
MAIL_MAILER=smtp
MAIL_HOST=127.0.0.1
MAIL_PORT=1025
```

----

## 4. Compatible with [Sentry](https://sentry.io/) reports

Buggregator can be used to receive Sentry reports from your application. It's a lightweight alternative for local
development. Just configure Sentry DSN to send data to Buggregator.

![sentry](https://user-images.githubusercontent.com/773481/208728578-1b33174b-8d1f-411a-a6fe-180a89abf06f.png)

### Spiral Framework

Spiral Framework is supported via a native package. You can read about integrations
on [official site](https://spiral.dev/docs/extension-sentry/3.3/en)

```dotenv
SENTRY_DSN=http://sentry@127.0.0.1:8000/1
```

### Laravel

Laravel is supported via a native package. You can read about integrations
on [official site](https://docs.sentry.io/platforms/php/guides/laravel/)

```dotenv
SENTRY_LARAVEL_DSN=http://sentry@127.0.0.1:8000/1
```

### Other platforms

To report to Buggregator you’ll need to use a language-specific SDK. The Sentry team builds and maintains these for most
popular languages.

You can find out documentation on [official site](https://docs.sentry.io/platforms/)

----

## 5. [Monolog](https://github.com/Seldaek/monolog) server

It can receive logs from `monolog/monolog` package via `\Monolog\Handler\SocketHandler` handler.

![monolog](https://user-images.githubusercontent.com/773481/208729325-b135870e-3a98-4841-90cb-6e507108a235.png)


### Spiral Framework

You can register socket handler for monolog via bootloader.

**Bootloader example**

```php
<?php

declare(strict_types=1);

namespace App\Bootloader;

use Monolog\Formatter\JsonFormatter;
use Monolog\Handler\SocketHandler;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Monolog\Bootloader\MonologBootloader;

class LoggingBootloader extends Bootloader
{
    public function init(MonologBootloader $monolog, EnvironmentInterface $env): void
    {
        $handler = new SocketHandler($env->get('MONOLOG_SOCKET_HOST'), chunkSize: 10);
        $handler->setFormatter(new JsonFormatter(JsonFormatter::BATCH_MODE_NEWLINES));
        $monolog->addHandler('socket', $handler);
    }
}
```

**Env variables**

```dotenv
MONOLOG_DEFAULT_CHANNEL=socket
MONOLOG_SOCKET_HOST=127.0.0.1:9913
```

### Laravel

**Config**

```php
// config/logging.php
return [
    // ...
    'channels' => [
        // ...
        'socket' => [
            'driver' => 'monolog',
            'level' => env('LOG_LEVEL', 'debug'),
            'handler' => \Monolog\Handler\SocketHandler::class,
            'formatter' => \Monolog\Formatter\JsonFormatter::class,
            'handler_with' => [
                'connectionString' => env('LOG_SOCKET_URL', '127.0.0.1:9913'),
            ],
        ],
    ],
];
```

**Env variables**

```dotenv
LOG_CHANNEL=socket
LOG_SOCKET_URL=127.0.0.1:9913
```

### Other PHP frameworks

Install monolog

```bash
composer require monolog/monolog
```

```php
<?php

use Monolog\Logger;
use Monolog\Handler\SocketHandler;
use Monolog\Formatter\JsonFormatter;

// create a log channel
$log = new Logger('buggregator');
$handler = new SocketHandler('127.0.0.1:9913');
$handler->setFormatter(new JsonFormatter());
$log->pushHandler($handler);

// Send records to the Buggregator
$log->warning('Foo');
$log->error('Bar');
```

----

## 6. Compatible with [Inspector](https://inspector.dev/) reports

Buggregator can be used to receive Inspector events from your application. It's a lightweight alternative for
local development. Just configure Inspector client URL to send data to Buggregator.

![inspector](https://user-images.githubusercontent.com/773481/208734651-e8dca2bf-6674-4aed-b6fc-601bc877f7ce.png)

### Laravel settings

Laravel is supported via a native package. You can read about integrations
on [official site](https://docs.inspector.dev/laravel)

```php
INSPECTOR_URL=http://127.0.0.1:8000/api/inspector
INSPECTOR_API_KEY=test
INSPECTOR_INGESTION_KEY=1test
INSPECTOR_ENABLE=true
```

### Other platforms

For PHP you can use `inspector-apm/inspector-php` package.

```php
use Inspector\Inspector;
use Inspector\Configuration;

$configuration = new Configuration('YOUR_INGESTION_KEY');
$configuration->setUrl('http://127.0.0.1:8000/api/inspector');
$inspector = new Inspector($configuration);

// ...
```

To report to Buggregator you’ll need to use a language-specific SDK. The Inspector team builds and maintains these for
most popular languages.

> Note: You can find out documentation on [official site](https://docs.inspector.dev/)

-----

## Technological stack

- [Spiral Framework](https://spiral.dev/)
- [RoadRunner](https://roadrunner.dev/) Http, Websocket, TCP, Queue, Cache server in one bottle
- [NuxtJs 3](https://nuxt.com/)
- [VueJs 3](https://v3.vuejs.org/)
- [TailwindCSS](https://tailwindcss.com/)
- [Storybook](https://storybook.js.org/)

## Installation

### Docker image

You can run Buggregator via docker from [Github Packages](https://github.com/buggregator/spiral-app/pkgs/container/server)

**Latest stable release**

```bash
docker run --pull always -p 8000:8000 -p 1025:1025 -p 9912:9912 -p 9913:9913 ghcr.io/buggregator/server:latest
```

**Latest dev release**

```bash
docker run --pull always -p 8000:8000 -p 1025:1025 -p 9912:9912 -p 9913:9913 ghcr.io/buggregator/server:dev
```

> Note: You can omit `--pull always` argument if your docker-compose doesn't support it.

**Specific version**

```bash
docker run -p 8000:8000 -p 1025:1025 -p 9912:9912 -p 9913:9913 ghcr.io/buggregator/server:v1.00
```

> Note: You can omit unused ports if you use, for example, only `var-dumper`

```bash
docker run --pull always -p 9912:9912 ghcr.io/buggregator/server:latest
```

### Using buggregator with docker compose

```yaml
version: "2"
services:
    # ...

    buggregator:
        image: ghcr.io/buggregator/server:dev
        ports:
            - 8000:8000
            - 1025:1025
            - 9912:9912
            - 9913:9913
```

That's it. Now you open http://127.0.0.1:8000 url in your browser and collect dumps from your application.

Enjoy!

---

# Contributing

There are several [issues](https://github.com/buggregator/spiral-app/issues) in this repo with unresolved issues, and it
would be great if you help a community solving them.

## Backend part

### Server requirements

1. PHP 8.1

### Installation

1. Clone repository `git clone https://github.com/buggregator/spiral-app.git`
2. Install backend dependencies `composer install`
3. Download RoadRunner binary `vendor/bin/rr get-binary`
4. Install frontend dependencies `cd frontend && yarn install`
5. Run RoadRunner server `./rr serve`

---

## License

Buggregator is open-sourced software licensed under the MIT license.
