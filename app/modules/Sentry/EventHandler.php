<?php

declare(strict_types=1);

namespace Modules\Sentry;

use App\Application\Event\EventType;
use Modules\Sentry\Application\DTO\Payload;
use Psr\Container\ContainerInterface;

final readonly class EventHandler implements Application\EventHandlerInterface
{
    /**
     * @param class-string<\Modules\Sentry\Application\EventHandlerInterface>[] $handlers
     */
    public function __construct(
        private ContainerInterface $container,
        private array $handlers,
    ) {}

    public function handle(Payload $payload, EventType $event): Payload
    {
        foreach ($this->handlers as $handler) {
            $payload = $this->container->get($handler)->handle($payload, $event);
        }

        return $payload;
    }
}
