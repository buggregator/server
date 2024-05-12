<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Webhooks\Domain\WebhookEvent;
use Modules\Webhooks\Domain\WebhookRepositoryInterface;
use Modules\Webhooks\Domain\WebhookServiceInterface;
use Modules\Webhooks\Interfaces\Job\JobPayload;
use Modules\Webhooks\Interfaces\Job\WebhookHandler;
use Psr\Log\LoggerInterface;
use Spiral\Queue\QueueInterface;

final readonly class WebhookService implements WebhookServiceInterface
{
    public function __construct(
        private WebhookRepositoryInterface $webhooks,
        private LoggerInterface $logger,
        private QueueInterface $queue,
    ) {}

    public function send(WebhookEvent $event): void
    {
        $found = $this->webhooks->findByEvent($event->event);

        foreach ($found as $webhook) {
            $this->sendWebhook($webhook->uuid, $event);
        }
    }

    private function sendWebhook(Uuid $uuid, WebhookEvent $event): void
    {
        $webhook = $this->webhooks->getByUuid($uuid);

        $this->logger->debug('Sending webhook', ['webhook' => $event->event, 'uuid' => (string) $webhook->uuid]);

        $this->queue->push(
            name: WebhookHandler::class,
            payload: new JobPayload(
                webhookUuid: $webhook->uuid->toObject(),
                event: $event,
            ),
        );
    }
}
