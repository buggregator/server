<?php

declare(strict_types=1);

namespace App\Application\Domain\ValueObjects;

final readonly class Json implements \JsonSerializable, \Stringable
{
    public function __construct(
        private array|\JsonSerializable $data = [],
    ) {}

    /**
     * Create from data storage raw value
     */
    final public static function typecast(mixed $value): self
    {
        if (empty($value)) {
            return new self();
        }

        try {
            return new self(
                (array) \json_decode($value, true),
            );
        } catch (\JsonException $e) {
            throw new \InvalidArgumentException($e->getMessage(), $e->getCode(), $e);
        }
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
