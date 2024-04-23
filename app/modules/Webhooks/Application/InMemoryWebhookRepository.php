<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use Modules\Webhooks\Domain\Webhook;
use Modules\Webhooks\Domain\WebhookRegistryInterface;
use Modules\Webhooks\Domain\WebhookRepositoryInterface;
use Modules\Webhooks\Exceptions\WebhookNotFoundException;
use Psr\Log\LoggerInterface;
use Ramsey\Uuid\UuidInterface;
use Spiral\Core\Attribute\Singleton;

#[Singleton]
final class InMemoryWebhookRepository implements WebhookRepositoryInterface, WebhookRegistryInterface
{
    /** @var Webhook[] */
    private array $webhooks = [];

    public function __construct(
        private LoggerInterface $logger,
    ) {
    }

    public function findByEvent(string $event): array
    {
        return \array_filter($this->webhooks, fn(Webhook $webhook) => $webhook->event === $event);
    }

    public function getByUuid(UuidInterface $uuid): Webhook
    {
        return $this->findByUuid($uuid) ?? throw new WebhookNotFoundException(
            \sprintf('Webhook with UUID %s not found', $uuid),
        );
    }

    public function findByUuid(UuidInterface $uuid): ?Webhook
    {
        return $this->webhooks[(string)$uuid] ?? null;
    }

    public function register(Webhook $webhook): void
    {
        $this->webhooks[(string)$webhook->uuid] = $webhook;

        $this->logger->info('Webhook registered', [
            'uuid' => (string)$webhook->uuid,
            'event' => $webhook->event,
            'url' => $webhook->url,
        ]);
    }
}
