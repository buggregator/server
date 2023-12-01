<?php

declare(strict_types=1);

namespace App\Application\Broadcasting;

use Spiral\Broadcasting\BroadcastInterface;
use Spiral\Core\CoreInterceptorInterface;
use Spiral\Core\CoreInterface;

final class BroadcastEventInterceptor implements CoreInterceptorInterface
{
    public function __construct(
        private readonly BroadcastInterface $broadcast
    ) {}

    public function process(string $controller, string $action, array $parameters, CoreInterface $core): mixed
    {
        $event = $parameters['event'];
        $result = $core->callAction($controller, $action, $parameters);

        if ($event instanceof ShouldBroadcastInterface) {
            $this->broadcast->publish(
                $event->getBroadcastTopics(),
                \json_encode([
                    'event' => $event->getEventName(),
                    'data' =>  $event->jsonSerialize(),
                ], JSON_THROW_ON_ERROR | JSON_UNESCAPED_UNICODE | JSON_INVALID_UTF8_SUBSTITUTE)
            );
        }

        return $result;
    }
}
