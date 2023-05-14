<?php

declare(strict_types=1);

namespace Modules\Smtp\Application;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Service\ClientProxy\EventHandlerInterface;
use Modules\Smtp\Application\Mail\Parser;
use Spiral\Cqrs\CommandBusInterface;

final class RequestHandler implements EventHandlerInterface
{
    public function __construct(
        private readonly Parser $parser,
        private readonly CommandBusInterface $commands,
    ) {
    }

    public function isSupported(string $type): bool
    {
        return $type === 'smtp';
    }

    public function handle(string $payload): void
    {
        $content = $this->parser
            ->parse($payload)
            ->storeAttachments()
            ->jsonSerialize();

        $this->commands->dispatch(
            new HandleReceivedEvent(type: 'smtp', payload: $content),
        );
    }
}
