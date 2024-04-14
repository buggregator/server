<?php

declare(strict_types=1);

namespace Modules\Ray;

use Modules\Ray\Application\EventHandlerInterface;
use Psr\Container\ContainerInterface;

final readonly class EventHandler implements EventHandlerInterface
{
    /**
     * @param class-string<EventHandlerInterface>[] $handlers
     */
    public function __construct(
        private ContainerInterface $container,
        private array $handlers,
    ) {
    }

    public function handle(array $event): array
    {
        foreach ($this->handlers as $handler) {
            $event = $this->container->get($handler)->handle($event);
        }

        return $event;
    }
}
