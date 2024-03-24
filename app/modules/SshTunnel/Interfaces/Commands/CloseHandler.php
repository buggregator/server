<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Interfaces\Commands;

use App\Application\Commands\ConnectSshTunnel;
use Modules\SshTunnel\Application\SshTunnelService;
use Modules\SshTunnel\Domain\ConnectionRepositoryInterface;
use Modules\SshTunnel\Exception\ConnectionException;
use Modules\SshTunnel\Exception\ConnectionNotFoundException;
use Spiral\Cqrs\Attribute\CommandHandler;

final class CloseHandler
{
    public function __construct(
        private readonly ConnectionRepositoryInterface $connections,
        private readonly SshTunnelService $service,
    ) {
    }

    #[CommandHandler]
    public function __invoke(ConnectSshTunnel $command): void
    {
        $connection = $this->connections->findByPK($command->connectionUuid);

        if ($connection === null) {
            throw new ConnectionNotFoundException('Connection not found');
        }

        try {
            $this->service->disconnect($connection);
        } catch (\Throwable $e) {
            throw new ConnectionException(previous: $e);
        }
    }
}
