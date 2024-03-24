<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Interfaces\Queries;

use App\Application\Commands\FindSshConnectionByUuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\SshTunnel\Domain\Connection;
use Modules\SshTunnel\Domain\ConnectionRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;

final class FindSshConnectionByUuidHandler
{
    public function __construct(
        private readonly ConnectionRepositoryInterface $connections,
    ) {
    }

    #[QueryHandler]
    public function __invoke(FindSshConnectionByUuid $query): Connection
    {
        $connection = $this->connections->findByPK((string)$query->uuid);
        if (!$connection) {
            throw new EntityNotFoundException(
                \sprintf('Connection with given uuid [%s] was not found.', (string)$query->uuid),
            );
        }

        return $connection;
    }
}
