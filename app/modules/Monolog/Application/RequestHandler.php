<?php

declare(strict_types=1);

namespace Modules\Monolog\Application;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Service\ClientProxy\EventHandlerInterface;
use Buggregator\Client\Proto\Frame;
use Spiral\Cqrs\CommandBusInterface;

final class RequestHandler implements EventHandlerInterface
{
    public function __construct(
        private readonly CommandBusInterface $commandBus,
    ) {
    }

    /**
     * @param Frame\Monolog $frame
     * @return void
     */
    public function handle(Frame $frame): void
    {
        $this->commandBus->dispatch(
            new HandleReceivedEvent(type: 'monolog', payload: $frame->message)
        );
    }

    public function isSupported(Frame $frame): bool
    {
        return $frame instanceof Frame\Monolog;
    }
}
