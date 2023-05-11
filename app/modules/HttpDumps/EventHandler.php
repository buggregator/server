<?php

declare(strict_types=1);

namespace Modules\HttpDumps;

use Modules\HttpDumps\Application;
use Psr\Container\ContainerInterface;

final class EventHandler implements Application\EventHandlerInterface
{
    /**
     * @param class-string<\Modules\Sentry\Application\EventHandlerInterface>[] $handlers
     */
    public function __construct(
        private readonly ContainerInterface $container,
        private readonly array $handlers
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
