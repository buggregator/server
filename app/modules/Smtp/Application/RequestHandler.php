<?php

declare(strict_types=1);

namespace Modules\Smtp\Application;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Service\ClientProxy\EventHandlerInterface;
use Buggregator\Client\Proto\Frame;
use Modules\Smtp\Application\Mail\Parser;
use Spiral\Cqrs\CommandBusInterface;

final class RequestHandler implements EventHandlerInterface
{
    public function __construct(
        private readonly Parser $parser,
        private readonly CommandBusInterface $commands,
    ) {
    }

    public function isSupported(Frame $frame): bool
    {
        return $frame instanceof Frame\Smtp;
    }

    /**
     * @param Frame\Smtp $frame
     */
    public function handle(Frame $frame): void
    {
        $content = $this->parser
            ->parse($frame->message)
            ->storeAttachments()
            ->jsonSerialize();

        $this->commands->dispatch(
            new HandleReceivedEvent(type: 'smtp', payload: $content),
        );
    }
}
