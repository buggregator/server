<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Domain;

interface ConnectionFactoryInterface
{
    public function make(
        string $name,
        string $host,
        string $user = 'root',
        int $port = 22,
        ?string $password = null,
        ?string $privateKey = null,
    ): Connection;
}
