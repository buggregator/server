<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Interfaces\Http\Controllers;

use App\Application\Commands\DeleteSshConnection;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\HTTP\Response\ResourceInterface;
use App\Application\HTTP\Response\SuccessResource;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Router\Annotation\Route;

final class DeleteAction
{
    #[Route(route: 'ssh/<uuid>', name: 'ssh.delete', methods: 'DELETE', group: 'api')]
    public function __invoke(CommandBusInterface $bus, Uuid $uuid): ResourceInterface
    {
        $bus->dispatch(
            new DeleteSshConnection(
                connectionUuid: $uuid,
            ),
        );

        return new SuccessResource();
    }
}
