# Clone the repository
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

RUN npm install
RUN npm run generate
RUN rm -rf node_modules

# Configure PHP project
FROM spiralscout/php81-grpc:1.0.0
COPY --from=frontend /app /app

ARG APP_VERSION=v1.0
ENV COMPOSER_ALLOW_SUPERUSER=1

RUN curl -s https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin/ --filename=composer
WORKDIR /app

RUN composer config --no-plugins allow-plugins.spiral/composer-publish-plugin false
RUN composer install --no-dev

RUN echo "Download RoadRunner"
RUN ./vendor/bin/rr get-binary

WORKDIR /app

RUN apk add --no-cache nginx
RUN cp docker/nginx/nginx.conf /etc/nginx/nginx.conf
RUN [ -d /etc/nginx/conf.d ] ||  mkdir /etc/nginx/conf.d
RUN cp docker/nginx/default.conf /etc/nginx/conf.d/default.conf

EXPOSE 80/tcp

LABEL org.opencontainers.image.source=$REPOSITORY
LABEL org.opencontainers.image.description="RoadRunner Birddog"
LABEL org.opencontainers.image.licenses=MIT

ENV RR_LOG_LEVEL=debug
ENV RR_CENTRIFUGE_NUM_WORKERS=5
ENV RR_HTTP_NUM_WORKERS=1
ENV RR_CENTRIFUGE_PROXY_ADDRESS="tcp://127.0.0.1:10001"
ENV RR_CENTRIFUGE_GRPC_API_ADDRESS="127.0.0.1:10000"

CMD ./rr serve -c .rr-prod.yaml
