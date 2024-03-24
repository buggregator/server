<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Spiral\Cqrs\CommandInterface;

final class StoreSshConnection implements CommandInterface
{
    public function __construct(
        public string $name,
        public string $host,
        public string $user = 'root',
        public int $port = 22,
        public ?string $password = null,
        public ?string $privateKey = null,
    ) {}
}
