<?php

declare(strict_types=1);

namespace Modules\Events\Domain;

use App\Application\Domain\Entity\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use DateTimeImmutable;

#[Entity(
    repository: EventRepositoryInterface::class
)]
class Event
{
    /**  @internal */
    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: 'uuid')]
        private Uuid $uuid,

        #[Column(type: 'string')]
        private string $type,

        #[Column(type: 'longText', typecast: 'json')]
        private Json $payload,

        #[Column(type: 'float')]
        private float $timestamp,

        #[Column(type: 'integer', nullable: true)]
        private ?int $projectId,
    ) {
    }

    public function getUuid(): Uuid
    {
        return $this->uuid;
    }

    public function getType(): string
    {
        return $this->type;
    }

    public function getPayload(): Json
    {
        return $this->payload;
    }

    public function getTimestamp(): float
    {
        return $this->timestamp;
    }

    public function getProjectId(): ?int
    {
        return $this->projectId;
    }
}
