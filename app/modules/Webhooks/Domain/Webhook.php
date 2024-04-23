<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use Psr\Http\Message\MessageInterface;
use Ramsey\Uuid\UuidInterface;

final class Webhook
{
    public function __construct(
        public UuidInterface $uuid,
        public string $event,
        public string $url,
        public array $headers = [],
        public bool $verifySsl = false,
        public bool $retryOnFailure = true,
    ) {
    }

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

    public function withHeader(string $name, $value): self
    {
        $clone = clone $this;
        $clone->headers[$name] = (array)$value;

        return $clone;
    }

    public function withAddedHeader(string $name, $value): self
    {
        $clone = clone $this;
        $clone->headers[$name][] = $value;

        return $clone;
    }

    public function withoutHeader(string $name): self
    {
        $clone = clone $this;
        unset($clone->headers[$name]);

        return $clone;
    }
}
