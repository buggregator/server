<?php

declare(strict_types=1);

namespace Modules\Projects\Domain\ValueObject;

use App\Application\Domain\Assert;

final readonly class Key implements \JsonSerializable, \Stringable
{
    public const MIN_LENGTH = 3;

    public const MAX_LENGTH = 36;

    public const ALLOWED_CHARACTERS = 'a-z0-9-_';

    public static function create(string $key): self
    {
        Assert::notEmpty(
            value: $key,
            message: 'Project key is required.',
        );

        Assert::minLength(
            value: $key,
            min: self::MIN_LENGTH,
            message: 'Invalid project key. Key must be at least 3 characters long.',
        );

        Assert::maxLength(
            value: $key,
            max: self::MAX_LENGTH,
            message: 'Invalid project key. Key must be less than 36 characters long.',
        );

        Assert::regex(
            value: $key,
            pattern: '/^[' . self::ALLOWED_CHARACTERS . ']+$/',
            message: 'Invalid project key. Key must contain only lowercase letters, numbers, hyphens and underscores.',
        );

        return new self($key);
    }

    /**
     * @internal
     * @private
     */
    public function __construct(
        public string $value,
    ) {}

    /**
     * Create from data storage raw value
     */
    final public static function typecast(mixed $value): self
    {
        return new self($value);
    }

    public function jsonSerialize(): string
    {
        return $this->value;
    }

    public function __toString(): string
    {
        return $this->value;
    }
}
