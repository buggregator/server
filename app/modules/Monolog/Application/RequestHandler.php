<?php

declare(strict_types=1);

namespace Modules\Monolog\Application;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Service\ClientProxy\EventHandlerInterface;
use Spiral\Cqrs\CommandBusInterface;

final class RequestHandler implements EventHandlerInterface
{
    public function __construct(
        private readonly CommandBusInterface $commandBus,
    ) {
    }

    public function handle(string $payload): void
    {
        $payload = \json_decode($payload, true, 512, JSON_THROW_ON_ERROR);

        if (!$payload) {
            throw new \RuntimeException("Unable to decode a message.");
        }

        $this->commandBus->dispatch(
            new HandleReceivedEvent(type: 'monolog', payload: $payload)
        );
    }

    public function isSupported(string $type): bool
    {
        return $type === 'monolog';
    }
}
