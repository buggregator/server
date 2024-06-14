<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\DTO;

final class BlobChunk implements PayloadChunkInterface
{
    public function __construct(
        public string $data,
    ) {}

    public function __toString(): string
    {
        return $this->data;
    }

    public function jsonSerialize(): string
    {
        return $this->data;
    }
}
