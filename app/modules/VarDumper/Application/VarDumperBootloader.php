<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application;

use Modules\VarDumper\Application\Dump\DumpIdGeneratorInterface;
use Modules\VarDumper\Application\Dump\MtRandDumpIdGenerator;
use Spiral\Boot\Bootloader\Bootloader;

final class VarDumperBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            DumpIdGeneratorInterface::class => MtRandDumpIdGenerator::class,
        ];
    }
}
