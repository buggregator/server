<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use App\Application\Domain\ValueObjects\Uuid;

final readonly class Delivery
{
    public \DateTimeInterface $createdAt;

    public function __construct(
        public Uuid $uuid,
        public Uuid $webhookUuid,
        public string $payload,
        public string $response,
        public int $status,
    ) {
        $this->createdAt = new \DateTimeImmutable();
    }
}
