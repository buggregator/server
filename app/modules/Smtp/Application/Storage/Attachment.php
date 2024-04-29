<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Storage;

final readonly class Attachment implements \JsonSerializable
{
    public function __construct(
        public string $name,
        public string $uri,
        public int $size,
        public string $mime,
        public string $id,
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'name' => $this->name,
            'uri' => $this->uri,
            'size' => $this->size,
            'mime' => $this->mime,
            'id' => $this->id,
        ];
    }
}

{
}
