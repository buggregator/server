<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use Ramsey\Uuid\UuidInterface;

interface WebhookRepositoryInterface
{
    /**
     * @param non-empty-string $event
     * @return Webhook[]
     */
    public function findByEvent(string $event): array;

    public function getByUuid(UuidInterface $uuid): Webhook;

    public function findByUuid(UuidInterface $uuid): ?Webhook;
}
