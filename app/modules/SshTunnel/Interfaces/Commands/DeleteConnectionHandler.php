<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Interfaces\Commands;

use App\Application\Commands\DeleteSshConnection;
use Modules\SshTunnel\Domain\ConnectionRepositoryInterface;
use Spiral\Cqrs\Attribute\CommandHandler;

final class DeleteConnectionHandler
{
    public function __construct(
        private readonly ConnectionRepositoryInterface $connections,
    ) {
    }

    #[CommandHandler]
    public function __invoke(DeleteSshConnection $command): void
    {
        $this->connections->deleteByPK($command->connectionUuid);
    }
}
