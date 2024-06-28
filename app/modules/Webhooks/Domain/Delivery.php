<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use Cycle\Annotated\Annotation\ForeignKey;

#[Entity(
    repository: DeliveryRepositoryInterface::class,
    table: 'webhook_deliveries',
)]
#[ForeignKey(target: Webhook::class, innerKey: 'webhook_uuid', outerKey: 'uuid')]
class Delivery
{
    #[Column(type: 'datetime')]
    private readonly \DateTimeInterface $createdAt;

    /**  @internal */
    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: 'uuid')]
        private readonly Uuid $uuid,
        #[Column(type: 'string(36)', typecast: 'uuid')]
        private readonly Uuid $webhookUuid,
        #[Column(type: 'text')]
        private readonly string $payload,
        #[Column(type: 'text')]
        private readonly string $response,
        #[Column(type: 'integer')]
        private readonly int $status,
    ) {
        $this->createdAt = new \DateTimeImmutable();
    }

    public function getUuid(): Uuid
    {
        return $this->uuid;
    }

    public function getWebhookUuid(): Uuid
    {
        return $this->webhookUuid;
    }

    public function getPayload(): string
    {
        return $this->payload;
    }

    public function getResponse(): string
    {
        return $this->response;
    }

    public function getStatus(): int
    {
        return $this->status;
    }

    public function getCreatedAt(): \DateTimeInterface
    {
        return $this->createdAt;
    }
}
