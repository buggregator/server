<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Webhooks\Domain\Webhook;
use Modules\Webhooks\Domain\WebhookRegistryInterface;
use Modules\Webhooks\Domain\WebhookRepositoryInterface;
use Modules\Webhooks\Exceptions\WebhookNotFoundException;
use Psr\Log\LoggerInterface;
use Psr\SimpleCache\CacheInterface;
use Spiral\Core\Attribute\Singleton;

#[Singleton]
final readonly class InMemoryWebhookRepository implements WebhookRepositoryInterface, WebhookRegistryInterface
{
    public function __construct(
        private LoggerInterface $logger,
        private CacheInterface $cache,
    ) {
    }

    public function findByEvent(string $event): array
    {
        return \array_filter($this->getWebhooks(), fn(Webhook $webhook) => $webhook->event === $event);
    }

    public function getByUuid(Uuid $uuid): Webhook
    {
        return $this->findByUuid($uuid) ?? throw new WebhookNotFoundException(
            \sprintf('Webhook with UUID %s not found', $uuid),
        );
    }

    public function findByUuid(Uuid $uuid): ?Webhook
    {
        return $this->getWebhooks()[(string)$uuid] ?? null;
    }

    public function register(Webhook $webhook): void
    {
        if ($this->findByUuid($webhook->uuid) !== null) {
            return;
        }

        $webhooks = $this->getWebhooks();
        $webhooks[(string)$webhook->uuid] = $webhook;

        $this->logger->debug('Webhook registered', [
            'uuid' => (string)$webhook->uuid,
            'event' => $webhook->event,
            'url' => $webhook->url,
        ]);

        $this->cache->set('webhooks', $webhooks);
    }

    /**
     * @return Webhook[]
     */
    private function getWebhooks(): array
    {
        return $this->cache->get('webhooks', []);
    }

    public function findAll(): array
    {
        return \array_values($this->getWebhooks());
    }
}
