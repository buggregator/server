<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Domain\Events;

use App\Application\Broadcasting\Channel\SettingsChannel;
use App\Application\Broadcasting\ShouldBroadcastInterface;
use Modules\SshTunnel\Domain\Connection;
use Stringable;

final class ConnectionStored implements ShouldBroadcastInterface
{
    public function __construct(
        public readonly Connection $connection,
    ) {
    }

    public function getEventName(): string
    {
        return 'ssh.connection.stored';
    }

    public function getBroadcastTopics(): iterable|string|Stringable
    {
        return new SettingsChannel();
    }

    public function jsonSerialize(): array
    {
        return [
            'name' => $this->connection->name,
            'host' => $this->connection->host,
            'user' => $this->connection->user,
            'port' => $this->connection->port,
            'privateKey' => $this->connection->privateKey,
        ];
    }
}
