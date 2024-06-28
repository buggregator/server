<?php

declare(strict_types=1);

namespace App\Application\Domain\ValueObjects;

use Ramsey\Uuid\UuidInterface;

final class Uuid implements \Stringable
{
    private readonly UuidInterface $uuid;

    public static function generate(): self
    {
        return new self();
    }

    public function __construct(?UuidInterface $uuid = null)
    {
        $this->uuid = $uuid ?? \Ramsey\Uuid\Uuid::uuid7();
    }

    public function equals(self $uuid): bool
    {
        return $this->uuid->equals($uuid->uuid);
    }

    public function toObject(): UuidInterface
    {
        return $this->uuid;
    }

    public static function fromString(string $uuid): self
    {
        return new self(\Ramsey\Uuid\Uuid::fromString($uuid));
    }

    public function __toString(): string
    {
        return $this->uuid->toString();
    }
}
