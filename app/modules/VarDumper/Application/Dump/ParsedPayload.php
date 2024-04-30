<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application\Dump;

use Symfony\Component\VarDumper\Cloner\Data;

final readonly class ParsedPayload
{
    public function __construct(
        public Data $data,
        public array $context,
    ) {}
}
