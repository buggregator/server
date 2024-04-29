<?php

declare(strict_types=1);

namespace App\Application\Broadcasting;

use Spiral\Broadcasting\BroadcastInterface;
use Spiral\Core\CoreInterceptorInterface;
use Spiral\Core\CoreInterface;

final readonly class BroadcastEventInterceptor implements CoreInterceptorInterface
{
    public function __construct(
        private BroadcastInterface $broadcast,
        private EventMapperInterface $mapper,
    ) {
    }

    public function process(string $controller, string $action, array $parameters, CoreInterface $core): mixed
    {
        $event = $parameters['event'];
        $result = $core->callAction($controller, $action, $parameters);

        if ($event instanceof ShouldBroadcastInterface) {
            $broadcastEvent = $this->mapper->toBroadcast($event);

            $this->broadcast->publish(
                $broadcastEvent->channel,
                \json_encode([
                    'event' => (string)$broadcastEvent->event,
                    'data' => $broadcastEvent->payload,
                ], JSON_THROW_ON_ERROR | JSON_UNESCAPED_UNICODE | JSON_INVALID_UTF8_SUBSTITUTE),
            );
        }

        return $result;
    }
}
