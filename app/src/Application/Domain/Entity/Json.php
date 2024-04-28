<?php

declare(strict_types=1);

namespace App\Application\Domain\Entity;

final readonly class Json implements \JsonSerializable, \Stringable
{
    public function __construct(
        private array|\JsonSerializable $data = [],
    ) {
    }

    /**
     * Create from data storage raw value
     */
    final public static function typecast(mixed $value): static
    {
        return new self(\json_decode($value, true));
    }

    public function jsonSerialize(): array
    {
        return $this->data instanceof \JsonSerializable
            ? $this->data->jsonSerialize()
            : $this->data;
    }

    public function __toString(): string
    {
        return \json_encode($this);
    }
}
