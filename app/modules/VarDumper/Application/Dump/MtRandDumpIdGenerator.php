<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application\Dump;

final readonly class MtRandDumpIdGenerator implements DumpIdGeneratorInterface
{
    public function generate(): string
    {
        return 'sf-dump-' . \mt_rand();
    }
}
