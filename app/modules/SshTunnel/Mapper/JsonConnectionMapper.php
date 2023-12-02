<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Mapper;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\SshTunnel\Domain\Connection;

final class JsonConnectionMapper
{
    public function fromArray(array $data): Connection
    {
        return new Connection(
            uuid: Uuid::fromString($data['uuid']),
            name: $data['name'],
            host: $data['host'],
            user: $data['user'],
            port: $data['port'],
            password: $data['password'],
            privateKey: $data['privateKey'],
        );
    }

    public function toArray(Connection $connection): array
    {
        return [
            'uuid' => (string)$connection->uuid,
            'name' => $connection->name,
            'host' => $connection->host,
            'user' => $connection->user,
            'port' => $connection->port,
            'password' => $connection->password,
            'privateKey' => $connection->privateKey,
        ];
    }
}
