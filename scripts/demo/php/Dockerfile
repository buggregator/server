# Demo VarDumper client: sends dump() calls to Buggregator over TCP.
#
#   docker build -t buggregator-demo-dumps scripts/demo/php
#   docker run --rm --add-host=host.docker.internal:host-gateway \
#     -e VAR_DUMPER_SERVER=host.docker.internal:9912 buggregator-demo-dumps
FROM composer:2 AS deps
WORKDIR /app
RUN composer require --no-interaction --no-progress --ignore-platform-reqs "symfony/var-dumper:^7.1"

FROM php:8.3-cli
WORKDIR /app
COPY --from=deps /app/vendor /app/vendor
COPY dumps.php /app/dumps.php
ENTRYPOINT ["php", "/app/dumps.php"]
