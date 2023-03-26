ARG ROAD_RUNNER_IMAGE=2.12.3

# Build rr binary
FROM ghcr.io/roadrunner-server/roadrunner:$ROAD_RUNNER_IMAGE as rr

# Clone the project
FROM alpine/git as git

ARG REPOSITORY=https://github.com/buggregator/spiral-app
ARG BRANCH=master
RUN git clone -b $BRANCH $REPOSITORY /app

WORKDIR /app/bin
# Download all required binaries
RUN chmod +x get-binaries.sh
RUN ./get-binaries.sh

# Build JS files
FROM node:19-alpine as frontend
COPY --from=git /app /app
WORKDIR /app/frontend

ENV NODE_OPTIONS=--openssl-legacy-provider

RUN yarn install
RUN npm run generate
RUN rm -rf node_modules

# Configure PHP project
FROM php:8.1.3-cli-alpine3.15 as backend

RUN apk add --no-cache \
        curl \
        libcurl \
        wget \
        libzip-dev \
        libmcrypt-dev \
        libxslt-dev \
        libxml2-dev \
        icu-dev \
        zip

RUN docker-php-ext-install \
        opcache \
        zip \
        xsl \
        dom \
        exif \
        intl \
        pcntl \
        bcmath \
        sockets

COPY --from=frontend /app /app
COPY --from=rr /usr/bin/rr /app

ARG APP_VERSION=v1.0
ENV COMPOSER_ALLOW_SUPERUSER=1

WORKDIR /app

RUN apk add --no-cache nginx
RUN cp docker/nginx/nginx.conf /etc/nginx/nginx.conf
RUN [ -d /etc/nginx/conf.d ] ||  mkdir /etc/nginx/conf.d
RUN cp docker/nginx/default.conf /etc/nginx/conf.d/default.conf

RUN curl -s https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin/ --filename=composer

RUN composer config --no-plugins allow-plugins.spiral/composer-publish-plugin false
RUN composer install --no-dev

RUN docker-php-source delete \
        && apk del ${BUILD_DEPENDS}

WORKDIR /app

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

LABEL org.opencontainers.image.source=$REPOSITORY
LABEL org.opencontainers.image.description="Buggregator"
LABEL org.opencontainers.image.licenses=MIT

CMD ./rr serve -c .rr-prod.yaml
