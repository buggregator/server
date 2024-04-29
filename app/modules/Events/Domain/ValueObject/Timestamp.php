<?php

declare(strict_types=1);

namespace Modules\Events\Domain\ValueObject;

final readonly class Timestamp implements \JsonSerializable, \Stringable
{
    public static function create(): self
    {
        return new self((string)\microtime(true));
    }

    /**
     * Create from data storage raw value
     */
    final public static function typecast(string $value): self
    {
        return new self($value);
    }

    /**
     * @internal
     * @private
     */
    public function __construct(
        private string $value,
    ) {
    }

    public function jsonSerialize(): float
    {
        return (float)$this->value;
    }

    public function __toString(): string
    {
        return $this->value;
    }
}
