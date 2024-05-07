<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application\Locator;

final class Webhook
{
    public function __construct(
        public readonly string $key,
        public readonly string $event,
        public readonly string $url,
        public array $headers = [],
        public readonly bool $verifySsl = false,
        public readonly bool $retryOnFailure = true,
    ) {}

    public function getHeaders(): array
    {
        return $this->headers;
    }

    public function hasHeader(string $name): bool
    {
        return isset($this->headers[$name]);
    }

    public function getHeader(string $name): array
    {
        return $this->headers[$name] ?? [];
    }

    public function getHeaderLine(string $name): string
    {
        return \implode(',', $this->headers[$name] ?? []);
    }

    public function withHeader(string $name, string|\Stringable|array $value): self
    {
        $clone = clone $this;
        $clone->headers[$name] = \array_map(\strval(...), (array) $value);

        return $clone;
    }
}
