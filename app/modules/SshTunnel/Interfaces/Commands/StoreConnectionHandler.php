<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Interfaces\Commands;

use App\Application\Commands\StoreSshConnection;
use Modules\SshTunnel\Domain\ConnectionFactoryInterface;
use Modules\SshTunnel\Domain\ConnectionRepositoryInterface;
use Modules\SshTunnel\Domain\Events\ConnectionStored;
use Psr\EventDispatcher\EventDispatcherInterface;
use Spiral\Cqrs\Attribute\CommandHandler;

final class StoreConnectionHandler
{
    public function __construct(
        private readonly EventDispatcherInterface $dispatcher,
        private readonly ConnectionRepositoryInterface $connections,
        private readonly ConnectionFactoryInterface $factory,
    ) {
    }

    #[CommandHandler]
    public function __invoke(StoreSshConnection $command): void
    {
        $connection = $this->factory->make(
            name: $command->name,
            host: $command->host,
            user: $command->user,
            port: $command->port,
            password: $command->password,
            privateKey: $command->privateKey,
        );

        $this->connections->store($connection);

        $this->dispatcher->dispatch(
            new ConnectionStored(connection: $connection),
        );
    }
}
