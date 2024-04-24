<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application\Dump;

interface DumpIdGeneratorInterface
{
    public function generate(): string;
}
