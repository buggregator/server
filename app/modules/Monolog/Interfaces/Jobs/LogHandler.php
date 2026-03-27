<?php

declare(strict_types=1);

namespace Modules\Monolog\Interfaces\Jobs;

use App\Application\Commands\HandleReceivedEvent;
use Psr\Log\LoggerInterface;
use Spiral\Core\InvokerInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Queue\JobHandler;

final class LogHandler extends JobHandler
{
    public function __construct(
        private readonly CommandBusInterface $commandBus,
        private readonly LoggerInterface $logger,
        InvokerInterface $invoker,
    ) {
        parent::__construct($invoker);
    }

    public function invoke(mixed $payload): void
    {
        if (\is_string($payload)) {
            $decoded = \json_decode($payload, true);
            if (\is_array($decoded) && isset($decoded['payload'])) {
                $monologPayload = $decoded['payload'];
                $project = $decoded['project'] ?? null;
            } else {
                $monologPayload = $decoded;
                $project = null;
            }
        } elseif (\is_array($payload)) {
            $monologPayload = $payload['payload'] ?? $payload;
            $project = $payload['project'] ?? null;
        } else {
            return;
        }

        if (!\is_array($monologPayload)) {
            $this->logger->error('[Monolog] Invalid payload format');
            return;
        }

        // Extract project from context if not already set
        if ($project === null) {
            $project = $monologPayload['context']['project'] ?? null;
        }

        if ($project !== null && !\is_string($project)) {
            $this->logger->warning('[Monolog] Project must be a string');
            $project = null;
        }

        $this->commandBus->dispatch(
            new HandleReceivedEvent(
                type: 'monolog',
                payload: $monologPayload,
                project: $project,
            ),
        );
    }
}
