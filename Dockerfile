# Stage 1: Build PHP VarDumper parser binary
FROM php:8.4-cli-alpine AS php-builder

RUN apk add --no-cache curl unzip
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

WORKDIR /php
COPY php/vardumper/composer.json php/vardumper/composer.lock* ./
RUN composer install --no-dev --optimize-autoloader --no-scripts

COPY php/vardumper/parser.php ./
COPY php/vardumper/box.json ./

# Download box.phar and build .phar
RUN curl -sL https://github.com/box-project/box/releases/download/4.6.6/box.phar -o box.phar \
    && php box.phar compile

# Download micro.sfx for the target architecture
ARG TARGETARCH
RUN ARCH=$(case ${TARGETARCH} in amd64) echo "x86_64";; arm64) echo "aarch64";; *) echo ${TARGETARCH};; esac) \
    && curl -sL "https://dl.static-php.dev/static-php-cli/common/php-8.4.19-micro-linux-${ARCH}.tar.gz" -o /tmp/micro.tar.gz \
    && mkdir -p /tmp/micro && tar -xzf /tmp/micro.tar.gz -C /tmp/micro/ \
    && cat /tmp/micro/micro.sfx vardumper-parser.phar > /php/vardumper-parser \
    && chmod +x /php/vardumper-parser

# Stage 2: Download frontend
FROM alpine:3.20 AS frontend
ARG FRONTEND_VERSION=1.29.1
RUN apk add --no-cache curl unzip \
    && mkdir -p /frontend \
    && curl -sL "https://github.com/buggregator/frontend/releases/download/${FRONTEND_VERSION}/frontend-${FRONTEND_VERSION}.zip" -o /tmp/fe.zip \
    && unzip -qo /tmp/fe.zip -d /frontend \
    && rm /tmp/fe.zip

# Stage 3: Build Go binary
FROM golang:1.26-alpine AS go-builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY cmd/ cmd/
COPY internal/ internal/
COPY modules/ modules/

# Copy built PHP binary into the embed location
COPY --from=php-builder /php/vardumper-parser modules/vardumper/bin/vardumper-parser-linux-amd64
# Also copy for arm64 (Docker picks the right one based on TARGETARCH)
COPY --from=php-builder /php/vardumper-parser modules/vardumper/bin/vardumper-parser-linux-arm64

# Copy frontend into the embed location
COPY --from=frontend /frontend/ internal/frontend/dist/

ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION}" -o buggregator ./cmd/buggregator

# Stage 4: Final minimal image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=go-builder /build/buggregator /usr/local/bin/buggregator

EXPOSE 8000 1025 9912 9913

ENTRYPOINT ["buggregator"]
