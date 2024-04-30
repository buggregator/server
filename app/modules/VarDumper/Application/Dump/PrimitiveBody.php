<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application\Dump;

final readonly class PrimitiveBody implements BodyInterface
{
    public function __construct(
        private string $type,
        private string $value,
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
        return $this->value;
    }

    public function jsonSerialize(): string
    {
        return $this->__toString();
    }
}
