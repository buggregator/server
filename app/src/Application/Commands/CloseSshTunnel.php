<?php

declare(strict_types=1);

namespace App\Application\Commands;

use App\Application\Domain\ValueObjects\Uuid;

final class CloseSshTunnel
{
    public function __construct(
        public readonly Uuid $connectionUuid,
    ) {}
}
