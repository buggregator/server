<?php

declare(strict_types=1);

namespace App\Application\Commands;

use App\Application\Domain\ValueObjects\Uuid;

final class ConnectSshTunnel
{
    public function __construct(
        public readonly Uuid $connectionUuid,
    ) {}
}
