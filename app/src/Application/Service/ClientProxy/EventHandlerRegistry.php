<?php

declare(strict_types=1);

namespace App\Application\Service\ClientProxy;

use Buggregator\Client\Proto\Frame;

final class EventHandlerRegistry implements EventHandlerRegistryInterface, CoreHandlerInterface
{
    /** @var EventHandlerInterface[] */
    private array $handlers = [];

    public function register(EventHandlerInterface $handler): void
    {
        $this->handlers[] = $handler;
    }

    public function handle(Frame $frame): void
    {
        foreach ($this->handlers as $handler) {
            if ($handler->isSupported($frame)) {
                $handler->handle($frame);
                return;
            }
        }

        throw new \RuntimeException('Unknown type of payload');
    }
}
