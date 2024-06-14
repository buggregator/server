<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\DTO;

readonly class JsonChunk implements PayloadChunkInterface
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
}
