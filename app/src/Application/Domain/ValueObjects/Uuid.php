<?php

declare(strict_types=1);

namespace App\Application\Domain\ValueObjects;

use Ramsey\Uuid\UuidInterface;

final class Uuid implements \Stringable
{
    public static function generate(): self
    {
        return new self();
    }

    public function __construct(private ?UuidInterface $uuid = null)
    {
        if (!$uuid) {
            $this->uuid = \Ramsey\Uuid\Uuid::uuid4();
        }
    }

    public function equals(self $uuid): bool
    {
        return $this->uuid->equals($uuid->uuid);
    }

    public function toObject(): UuidInterface
    {
        return $this->uuid;
    }

    public static function fromString(string $aggregateRootId): self
    {
        return new self(\Ramsey\Uuid\Uuid::fromString($aggregateRootId));
    }

    public function __toString()
    {
        return $this->uuid->toString();
    }
}
