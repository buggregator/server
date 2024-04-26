<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use App\Application\Domain\ValueObjects\Uuid;

interface WebhookRepositoryInterface
{
    /**
     * @param non-empty-string $event
     * @return Webhook[]
     */
    public function findByEvent(string $event): array;

    public function getByUuid(Uuid $uuid): Webhook;

    public function findByUuid(Uuid $uuid): ?Webhook;

    /**
     * @return Webhook[]
     */
    public function findAll(): array;
}
