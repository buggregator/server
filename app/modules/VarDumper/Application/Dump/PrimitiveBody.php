<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application\Dump;

final readonly class PrimitiveBody implements BodyInterface
{
    public function __construct(
        private string $type,
        private mixed $value,
    ) {}

    public function getType(): string
    {
        return $this->type;
    }

    public function getValue(): string
    {
        return $this->value;
    }

    public function __toString(): string
    {
        return match (true) {
            $this->value === true => '1',
            $this->value === false => '0',
            default => (string) $this->value,
        };
    }

    public function jsonSerialize(): string
    {
        return (string) $this;
    }
}
