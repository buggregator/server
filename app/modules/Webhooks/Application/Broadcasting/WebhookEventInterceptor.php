<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application\Broadcasting;

use Modules\Events\Domain\Events\EventWasReceived;
use Modules\Webhooks\Domain\WebhookEvent;
use Modules\Webhooks\Domain\WebhookServiceInterface;
use Spiral\Core\CoreInterceptorInterface;
use Spiral\Core\CoreInterface;

final readonly class WebhookEventInterceptor implements CoreInterceptorInterface
{
    public function __construct(
        private WebhookServiceInterface $service,
    ) {
    }

    public function process(string $controller, string $action, array $parameters, CoreInterface $core): mixed
    {
        $event = $parameters['event'];

        $result = $core->callAction($controller, $action, $parameters);

        $webhookEvent = null;
        if ($event instanceof EventWasReceived) {
            $webhookEvent = new WebhookEvent(
                event: $event->event->getType() . '.received',
                payload: [
                    'uuid' => (string)$event->event->getUuid(),
                    'type' => $event->event->getType(),
                    'payload' => $event->event->getPayload(),
                    'timestamp' => $event->event->getTimestamp(),
                ],
            );
        }


        if ($webhookEvent) {
            $this->service->send($webhookEvent);
        }

        return $result;
    }
}
