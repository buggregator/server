<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\DTO;

readonly class JsonChunk implements PayloadChunkInterface, \ArrayAccess
{
    public function __construct(
        public array $data,
    ) {}

    public function __toString(): string
    {
        return \json_encode($this->data, \JSON_THROW_ON_ERROR);
    }

    public function jsonSerialize(): array
    {
        return $this->data;
    }

    public function offsetExists(mixed $offset): bool
    {
        return isset($this->data[$offset]);
    }

    public function offsetGet(mixed $offset): mixed
    {
        return $this->data[$offset];
    }

    public function offsetSet(mixed $offset, mixed $value): void
    {
        throw new \BadMethodCallException('JsonChunk is readonly');
    }

    public function offsetUnset(mixed $offset): void
    {
        throw new \BadMethodCallException('JsonChunk is readonly');
    }
}
