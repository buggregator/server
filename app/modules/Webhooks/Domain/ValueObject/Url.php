<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain\ValueObject;

use App\Application\Exception\InvalidArgumentException;

final readonly class Url implements \JsonSerializable, \Stringable
{
    /**
     * Create from data storage raw value
     */
    final public static function typecast(string $value): self
    {
        return new self($value);
    }

    public static function create(string $value): self
    {
        \filter_var($value, \FILTER_VALIDATE_URL) ?: throw new InvalidArgumentException('Invalid URL');

        if (!\str_starts_with($value, 'http://') && !\str_starts_with($value, 'https://')) {
            throw new InvalidArgumentException('URL must start only with http:// or https://');
        }

        return new self($value);
    }

    public function __construct(
        private string $value,
    ) {}

    public function jsonSerialize(): string
    {
        return (string) $this;
    }

    public function __toString(): string
    {
        return $this->value;
    }
}
