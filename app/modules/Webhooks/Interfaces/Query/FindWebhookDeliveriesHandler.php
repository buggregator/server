<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Query;

use App\Application\Commands\FindWebhookDeliveries;
use Modules\Webhooks\Domain\DeliveryRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;

final readonly class FindWebhookDeliveriesHandler
{
    public function __construct(
        private DeliveryRepositoryInterface $deliveries,
    ) {}

    #[QueryHandler]
    public function __invoke(FindWebhookDeliveries $query): iterable
    {
        return $this->deliveries->findByWebhook(
            webhookUuid: $query->webhookUuid,
        );
    }
}
