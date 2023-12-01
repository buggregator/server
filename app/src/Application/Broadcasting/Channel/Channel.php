<?php

declare(strict_types=1);

namespace App\Application\Broadcasting\Channel;

class Channel implements \Stringable
{
    public function __construct(
        public readonly string $name
    ) {}

    public function __toString(): string
    {
        return $this->name;
    }
}
