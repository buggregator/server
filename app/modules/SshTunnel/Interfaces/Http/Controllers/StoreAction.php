<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Interfaces\Http\Controllers;

use App\Application\Commands\StoreSshConnection;
use App\Application\HTTP\Response\ResourceInterface;
use App\Application\HTTP\Response\SuccessResource;
use Modules\SshTunnel\Interfaces\Http\Request\StoreRequest;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Router\Annotation\Route;

final class StoreAction
{
    #[Route(route: 'ssh/connection', name: 'ssh.store', methods: 'POST', group: 'api')]
    public function __invoke(StoreRequest $request, CommandBusInterface $bus): ResourceInterface
    {
        $bus->dispatch(
            new StoreSshConnection(
                name: $request->name,
                host: $request->host,
                user: $request->user,
                port: $request->port,
                privateKey: $request->privateKey,
            ),
        );

        return new SuccessResource();
    }
}
