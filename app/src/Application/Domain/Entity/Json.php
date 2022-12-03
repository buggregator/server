<?php

declare(strict_types=1);

namespace App\Application\Domain\Entity;

use Cycle\Database\DatabaseInterface;

final class Json implements \JsonSerializable
{
    public function __construct(
        private readonly array $data = []
    ) {
    }

    public static function cast(string $value, DatabaseInterface $db): self
    {
        return new self(\json_decode($value, true));
    }

    public function jsonSerialize(): array
    {
        return $this->data;
    }
}
