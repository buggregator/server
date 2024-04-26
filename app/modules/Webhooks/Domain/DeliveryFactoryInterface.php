<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use App\Application\Domain\ValueObjects\Uuid;

interface DeliveryFactoryInterface
{
    public function create(
        Uuid $webhookUuid,
        string $payload,
        string $response,
        int $status,
    ): Delivery;
}
