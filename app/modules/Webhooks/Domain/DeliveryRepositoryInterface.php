<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use App\Application\Domain\ValueObjects\Uuid;

interface DeliveryRepositoryInterface
{
    public function store(Delivery $delivery): void;

    /**
     * @return Delivery[]
     */
    public function findAll(Uuid $webhookUuid): array;
}
