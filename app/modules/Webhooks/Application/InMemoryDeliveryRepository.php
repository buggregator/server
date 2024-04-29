<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Webhooks\Domain\Delivery;
use Modules\Webhooks\Domain\DeliveryRepositoryInterface;
use Psr\SimpleCache\CacheInterface;
use Spiral\Core\Attribute\Singleton;

#[Singleton]
final readonly class InMemoryDeliveryRepository implements DeliveryRepositoryInterface
{
    public function __construct(
        private CacheInterface $cache,
        private int $maxDeliveries = 10,
        private int $ttl = 3600, // 1 hour in seconds
    ) {
    }

    public function findAll(Uuid $webhookUuid): array
    {
        return $this->cache->get($this->getKey($webhookUuid), []);
    }

    public function store(Delivery $delivery): void
    {
        $deliveries = \array_slice(
            $this->cache->get($this->getKey($delivery->webhookUuid), []),
            0,
            $this->maxDeliveries,
        );

        $deliveries[] = $delivery;

        $this->cache->set($this->getKey($delivery->webhookUuid), $deliveries, $this->ttl);
    }

    /**
     * @return non-empty-string
     */
    public function getKey(Uuid $webhookUuid): string
    {
        return 'deliveries:' . $webhookUuid;
    }
}
