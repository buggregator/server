ARG ROAD_RUNNER_IMAGE=2.12.3
ARG CENTRIFUGO_IMAGE=v4

# Build centrifugo binary
FROM centrifugo/centrifugo:$CENTRIFUGO_IMAGE as centrifugo

# Build rr binary
FROM ghcr.io/roadrunner-server/roadrunner:$ROAD_RUNNER_IMAGE as rr

# Build JS files
FROM ghcr.io/buggregator/frontend:latest as frontend

# Clone the project
FROM alpine/git as git

ARG REPOSITORY=https://github.com/buggregator/server
ARG BRANCH=master
RUN git clone -b $BRANCH $REPOSITORY /app

# Configure PHP project
FROM ghcr.io/buggregator/docker:latest as backend

COPY --from=git /app /app
COPY --from=frontend /app /app/frontend
COPY --from=rr /usr/bin/rr /app
COPY --from=centrifugo /usr/local/bin/centrifugo /app/bin

ARG APP_VERSION=v1.0
ENV COMPOSER_ALLOW_SUPERUSER=1

RUN sed -i 's/memory_limit = 128M/memory_limit = 1024M/g' "$PHP_INI_DIR/php.ini-production" && \
    sed -i 's/post_max_size = 8M/post_max_size = 1024M/g' "$PHP_INI_DIR/php.ini-production" && \
    mv "$PHP_INI_DIR/php.ini-production" "$PHP_INI_DIR/php.ini"

WORKDIR /app

RUN composer config --no-plugins allow-plugins.spiral/composer-publish-plugin false
RUN composer install --no-dev

RUN docker-php-source delete \
        && apk del ${BUILD_DEPENDS}

WORKDIR /app
RUN php app.php configure

EXPOSE 8000/tcp

ENV RR_LOG_LEVEL=error
ENV RR_HTTP_NUM_WORKERS=2
ENV RR_CENTRIFUGE_NUM_WORKERS=3
ENV RR_HTTP_NUM_WORKERS=2
ENV RR_TCP_NUM_WORKERS=2
ENV RR_CENTRIFUGE_PROXY_ADDRESS="tcp://127.0.0.1:10001"
ENV RR_CENTRIFUGE_GRPC_API_ADDRESS="127.0.0.1:10000"

ENV BROADCAST_CONNECTION=centrifugo
ENV QUEUE_CONNECTION=roadrunner
ENV MONOLOG_DEFAULT_CHANNEL=roadrunner
ENV MONOLOG_DEFAULT_LEVEL=ERROR
ENV APP_ENV=production
ENV DEBUG=false
ENV ENCRYPTER_KEY=def00000232ae92c8e8ec0699093fa06ce014cd48d39c3c62c279dd947db084e56ee48b5c91cebc1c5abe53f7755021d09043757561c244c1c0c765cfeb5db33eb45a903
ENV TOKENIZER_CACHE_TARGETS=true

LABEL org.opencontainers.image.source=$REPOSITORY
LABEL org.opencontainers.image.description="Buggregator"
LABEL org.opencontainers.image.licenses=MIT

CMD ./rr serve -c .rr-prod.yaml
