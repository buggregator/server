<?php

declare(strict_types=1);

namespace App\Application\Service\ClientProxy;

final class EventHandlerRegistry implements EventHandlerRegistryInterface, CoreHandlerInterface
{
    /** @var EventHandlerInterface[] */
    private array $handlers = [];

    public function register(EventHandlerInterface $handler): void
    {
        $this->handlers[] = $handler;
    }

    public function handle(string $type, string $payload): void
    {
        foreach ($this->handlers as $handler) {
            if ($handler->isSupported($type)) {
                $handler->handle($payload);
                return;
            }
        }

        throw new \RuntimeException('Unknown type of payload');
    }
}
