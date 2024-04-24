<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Webhooks\Domain\Delivery;
use Modules\Webhooks\Domain\DeliveryFactoryInterface;

final readonly class DeliveryFactory implements DeliveryFactoryInterface
{
    public function create(
        Uuid $webhookUuid,
        string $payload,
        string $response,
        int $status,
    ): Delivery {
        return new Delivery(
            uuid: Uuid::generate(),
            webhookUuid: $webhookUuid,
            payload: $payload,
            response: $response,
            status: $status,
        );
    }
}
