<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application\Dump;

interface BodyInterface extends \Stringable, \JsonSerializable
{
    public function getType(): string;

    public function getValue(): string;
}
