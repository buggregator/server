# A server for debugging PHP applications and more.

[![Support me on Patreon](https://img.shields.io/endpoint.svg?url=https%3A%2F%2Fshieldsio-patreon.vercel.app%2Fapi%3Fusername%3Dbutschster%26type%3Dpatrons&style=flat)](https://patreon.com/butschster)
[![Downloads](https://img.shields.io/docker/pulls/butschster/buggregator.svg)](https://hub.docker.com/repository/docker/butschster/buggregator)
[![Twitter](https://img.shields.io/badge/twitter-Follow-blue)](https://twitter.com/buggregator)

![Cover image](https://user-images.githubusercontent.com/773481/208718792-eeae35a6-c5a8-4be4-9474-2b96d222e750.png)

Buggregator is a lightweight, standalone server that offers a range of debugging features for PHP applications. Built
using the reliable [Spiral Framework](https://spiral.dev/), the powerful [NuxtJs](https://nuxt.com/), and the
speedy [RoadRunner](https://roadrunner.dev/) under the hood. It's a versatile tool that can be run on multiple
platforms via Docker.

Whether you're an experienced developer or just starting, Buggregator offers essential features like
[Xhprof](#1-compatible-with-xhprof), [Symfony var-dumper](#2-symfony-vardumper-server), [Monolog](#3-monolog-server),
[Sentry](#4-compatible-with-sentry), [SMTP catcher](#5-fake-smtp-server-for-catching-mail),
and [Inspector](#6-compatible-with-inspector) that make it easy to identify and resolve issues. With no additional
packages required, it's effortless to use, making it a must-have tool in your development arsenal.

#### Contents

1. [Features](#features)
    - [Xhprof profiler](#1-xhprof-profiler)
    - [Symfony VarDumper server](#2-symfony-vardumper-server)
    - [Fake SMTP server](#3-fake-smtp-server-for-catching-mail)
    - [Sentry server](#4-compatible-with-sentry-reports)
    - [Monolog server](#5-monolog-server)
    - [Inspector](#6-compatible-with-inspector-reports)
    - [HTTP Requests dump server](#7-http-requests-dump-server)
    - [Spatie Ray debug tool](#8-spatie-ray-debug-tool)
2. [Installation](#installation)
    - [Docker image](#docker-image)
    - [Docker compose](#docker-compose)
3. [Configuration](#configuration)
4. [Contributing](#contributing)
5. [License](#license)

# Features

In this section, we'll explore the different features that Buggregator supports and how they can help you identify and
resolve issues with your application.

## 1. Xhprof profiler

The Xhprof profiler is an essential feature of Buggregator that offers a lightweight and hierarchical profiling solution
for PHP applications. It uses instrumentation to keep track of call counts and inclusive metrics for arcs in the dynamic
callgraph of your program during the data collection phase. In the reporting and post-processing phase, the profiler
computes exclusive metrics such as wall (elapsed) time, CPU time, and memory usage.

With the Xhprof profiler, you can easily identify performance bottlenecks and optimize your application's code for
better efficiency. So, if you're looking to fine-tune your PHP application's performance, the Xhprof profiler is the
perfect tool for the job.

![xhprof](https://user-images.githubusercontent.com/773481/208724383-3790a3e1-9ebe-4616-8d4d-d1869f8f2b7c.png)

### Installation

1. Install Xhprof extension
   One of the way to install Xhprof is to use [PECL](https://pecl.php.net/package/xhprof) package.

```bash
pear channel-update pear.php.net
pecl install xhprof
```

2. Install the package

If you are using Spiral Framework you just need to install
the [spiral/profiler](https://github.com/spiral/profiler/tree/3.0) package.

```bash
composer require --dev spiral/profiler:^3.0
```

### Configuration

After installing the package, you need to configure it. The package provides predefined environment variables to
configure the profiler.

```dotenv
PROFILER_ENDPOINT=http://profiler@127.0.0.1:8000
PROFILER_APP_NAME=My super app
```

> **Note:**
> Read more about package usage in the [package documentation](https://github.com/spiral/profiler/tree/3.0).

## 2. Symfony VarDumper server

Buggregator is fully compatible with the
Symfony [VarDumper component](https://symfony.com/doc/current/components/var_dumper.html#the-dump-server), which is an
essential feature for debugging PHP applications. By default, the `dump()` and `dd()` functions output their contents
in the same browser window or console terminal as your own application. This can be confusing at times, as it mixes the
real output with the debug output.

With Buggregator, however, you can easily collect all the dumped data, making it simpler to identify and isolate issues.
So, whether you're an experienced developer or just starting, the Symfony VarDumper server in Buggregator is an
essential tool to have in your development arsenal.

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

----

## 3. Fake SMTP server for catching mail

Buggregator is more than just a PHP debugging tool. It also includes a powerful email testing feature that allows you to
install and configure a local email server with ease.

For example, you can configure a local WordPress site to use Buggregator's SMTP server for email deliveries. This makes
it effortless to test email functionality during the development phase, ensuring that everything works as expected
before deployment. So, if you're looking for a reliable and easy-to-use email testing tool, Buggregator's fake SMTP
server is the way to go.

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

Buggregator offers seamless integration with Sentry reports, making it a reliable tool for local development. With
it, you can easily configure your Sentry DSN to send data directly to the server, providing you with a lightweight
alternative for debugging your application.

By using Buggregator to receive Sentry reports, you can identify and fix issues with your application before deploying
it to production. This ensures that your application is robust and efficient, providing a smooth experience for your
users. So, if you're looking for an easy and efficient way to receive Sentry reports during local development,
Buggregator is the perfect tool for you.

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

Buggregator comes with a powerful Monolog server that can receive logs from the popular `monolog/monolog` package via
the `\Monolog\Handler\SocketHandler` handler. With this feature, you can easily track and analyze the logs generated by
your PHP application, making it easier to identify issues and improve its overall performance.

By using Buggregator's Monolog server, you can gain valuable insights into your application's behavior and improve its
overall efficiency. So, whether you're a seasoned developer or just starting, the Monolog server in Buggregator is a
must-have tool for anyone serious about PHP development.

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

Buggregator is also compatible with Inspector reports, providing you with a lightweight alternative for local
development. With it, you can easily configure your Inspector client URL to send data directly to the server, making it
easier to identify and fix issues during the development phase.

![inspector](https://user-images.githubusercontent.com/773481/208734651-e8dca2bf-6674-4aed-b6fc-601bc877f7ce.png)

### Laravel settings

Laravel is supported via a native package. You can read about integrations
on [official site](https://docs.inspector.dev/laravel)

```php
INSPECTOR_URL=http://inspector@127.0.0.1:8000
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
$configuration->setUrl('http://inspector@127.0.0.1:8000');
$inspector = new Inspector($configuration);

// ...
```

To report to Buggregator you’ll need to use a language-specific SDK. The Inspector team builds and maintains these for
most popular languages.

> **Note:**
> You can find out documentation on [official site](https://docs.inspector.dev/)

## 7. HTTP Requests dump server

It's an indispensable tool that simplifies the process of capturing, analyzing, and debugging HTTP requests in their
applications. With the HTTP Requests Dump Server, developers can effortlessly capture all the relevant request data and
gain valuable insights. They can dive deep into the captured requests, examine their contents, and pinpoint any issues
or anomalies that might be affecting their application's performance.

Simply start the server and send your requests to the `http://http-dump@127.0.0.1:8000` URL, it will not only
capture the URI segments but also gather additional details such as the request `headers`, `cookies`, `POST data`,
`query strings`, and any `uploaded files`.

For instance, let's say you have a POST request: `http://http-dump@127.0.0.1:8000/user/3/update`. In this case,
server will intercept the request and capture all the relevant information. It will then display the
dumped data, allowing you to examine the request details, including the URI segments (`user/3/update` in this example).

![HTTP Requests dump server](https://github.com/spiral/docs/assets/773481/209f9c8c-00d2-4086-9f54-ce2cf8121394)

-----

## 8. Spatie [Ray debug tool](https://github.com/spatie/ray)

Buggregator is compatible with `spatie/ray` package. The Ray debug tool supports PHP, Ruby, JavaScript, TypeScript,
NodeJS, Go and Bash applications. After installing one of the libraries, you can use the ray function to quickly dump
stuff. Any variable(s) that you pass will be sent to the Buggregator.

![Ray debug tool](https://github.com/buggregator/spiral-app/assets/773481/c2a84d40-fc99-4bde-b87f-ea81cc1daa17)

**Supported features**: Simple data, Labels, Caller, Trace, Counter, Class name of an object, Measure, Json, Xml,
Carbon, File, Table, Image, Html, Text, Notifications, Phpinfo, Exception, Show queries, Count queries, Show events,
Show jobs, Show cache, Model, Show views, Markdown, Collections, Env, Response, Request, Application log, Show Http
client requests

### Laravel settings

Please make sure `ray.php` config published to the project root.

You can run an artisan command to publish it in to the project root.

```bash
php artisan ray:publish-config
```

**Env variables**

```
RAY_HOST=ray@127.0.0.1  # Ray server host (Current HTTP buggregator port)
RAY_PORT=8000           # Ray server port
```

### Framework agnostic PHP

In framework agnostic projects you can use this template as the ray config file.

```php
<?php
// Save this in a file called "ray.php"

return [
    /*
    * This settings controls whether data should be sent to Ray.
    */
    'enable' => true,

    /*
     *  The host used to communicate with the Ray app.
     */
    'host' => 'ray@127.0.0.1',

    /*
     *  The port number used to communicate with the Ray app.
     */
    'port' => 8000,

    /*
     *  Absolute base path for your sites or projects in Homestead, Vagrant, Docker, or another remote development server.
     */
    'remote_path' => null,

    /*
     *  Absolute base path for your sites or projects on your local computer where your IDE or code editor is running on.
     */
    'local_path' => null,

    /*
     * When this setting is enabled, the package will not try to format values sent to Ray.
     */
    'always_send_raw_values' => false,
];
```

You can find out more information about installation and configuration
on [official site](https://spatie.be/docs/ray/v1/introduction)

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

To run Buggregator using docker, you can use the docker image available
on [Github Packages](https://github.com/buggregator/spiral-app/pkgs/container/server)

**Latest stable release**

To run the latest stable release, use the following command:

```bash
docker run --pull always -p 8000:8000 -p 1025:1025 -p 9912:9912 -p 9913:9913 ghcr.io/buggregator/server:latest
```

**Latest dev release**

To run the latest development release, use the following command:

```bash
docker run --pull always -p 8000:8000 -p 1025:1025 -p 9912:9912 -p 9913:9913 ghcr.io/buggregator/server:dev
```

> **Note:**
> You can omit `--pull always` argument if your docker-compose doesn't support it.

**Specific version**

To run a specific version of Buggregator, use the following command:

```bash
docker run -p 8000:8000 -p 1025:1025 -p 9912:9912 -p 9913:9913 ghcr.io/buggregator/server:v1.00
```

> **Note:**
> You can omit unused ports if you only use, for example, `symfony/var-dumper`.

```bash
docker run --pull always -p 9912:9912 ghcr.io/buggregator/server:latest
```

### Using buggregator with docker compose

You can also use Buggregator with Docker Compose. Add the following service definition to
your `docker-compose.yml` file:

```yaml
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

After that, you can open http://127.0.0.1:8000 in your browser and start collecting dumps from your application.

Enjoy!

---

# Contributing

Contributing to open source projects like Buggregator can be a rewarding experience, and we welcome contributions from
the community. There are several [issues](https://github.com/buggregator/spiral-app/issues) in this repo with unresolved
issues, and it would be great if you help a community solving them.

**We appreciate any contributions to help make Buggregator better!***

## Backend part

### Server requirements

1. PHP 8.1

### Installation

1. Clone repository `git clone https://github.com/buggregator/spiral-app.git`
2. Install backend dependencies `composer install`
3. Download RoadRunner binary `vendor/bin/rr get-binary`
4. Install frontend dependencies `cd frontend && yarn install`
5. Install Centrifugo server `cd bin && ./get-binaries.sh`
6. Run RoadRunner server `./rr serve`

---

## License

Buggregator is open-sourced software licensed under the MIT license.
