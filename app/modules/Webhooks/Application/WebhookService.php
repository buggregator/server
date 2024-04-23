<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use Modules\Webhooks\Domain\WebhookEvent;
use Modules\Webhooks\Domain\WebhookRepositoryInterface;
use Modules\Webhooks\Domain\WebhookServiceInterface;
use Modules\Webhooks\Interfaces\Job\JobPayload;
use Modules\Webhooks\Interfaces\Job\WebhookHandler;
use Psr\Log\LoggerInterface;
use Ramsey\Uuid\UuidInterface;
use Spiral\Queue\QueueConnectionProviderInterface;
use Spiral\Queue\QueueInterface;

final readonly class WebhookService implements WebhookServiceInterface
{
    private QueueInterface $queue;

    public function __construct(
        private WebhookRepositoryInterface $webhooks,
        private LoggerInterface $logger,
        QueueConnectionProviderInterface $provider,
    ) {
        $this->queue = $provider->getConnection('webhook');
    }

    public function send(WebhookEvent $event): void
    {
        $found = $this->webhooks->findByEvent($event->event);

        foreach ($found as $webhook) {
            $this->sendWebhook($webhook->uuid, $event);
        }
    }

    public function sendWebhook(UuidInterface $uuid, WebhookEvent $event): void
    {
        $webhook = $this->webhooks->getByUuid($uuid);

        $this->logger->debug('Sending webhook', ['webhook' => $event->event, 'uuid' => (string) $webhook->uuid]);

        $this->queue->push(
            WebhookHandler::class,
            new JobPayload(
                $webhook->uuid,
                $event->event,
                $event->payload,
            ),
        );
    }
}
