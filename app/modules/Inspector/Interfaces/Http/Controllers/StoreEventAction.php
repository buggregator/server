<?php

declare(strict_types=1);

namespace Modules\Inspector\Interfaces\Http\Controllers;

use App\Application\Commands\HandleReceivedEvent;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\Exception\ClientException;
use Spiral\Router\Annotation\Route;

final class StoreEventAction
{
    #[Route(route: 'inspector', name: 'inspector.event.store', methods: ['POST'], group: 'api')]
    public function __invoke(
        ServerRequestInterface $request,
        CommandBusInterface $commands,
        int $projectId,
    ): void {
        $data = \json_decode(\base64_decode((string)$request->getBody()), true)
            ?? throw new ClientException\BadRequestException('Invalid data');

        $type = $data[0]['type'] ?? 'unknown';

        if ($type !== 'request') {
            throw new ClientException\BadRequestException('Invalid data');
        }

        $commands->dispatch(
            new HandleReceivedEvent(type: 'sentry', payload: $data)
        );
    }
}
