<?php

declare(strict_types=1);

namespace Modules\Events\Domain;

use App\Application\Domain\ValueObjects\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use Cycle\Annotated\Annotation\Table\Index;
use Modules\Events\Domain\ValueObject\Timestamp;
use Modules\Projects\Domain\ValueObject\Key;

#[Entity(
    repository: EventRepositoryInterface::class,
)]
#[Index(columns: ['type'])]
#[Index(columns: ['project'])]
#[Index(columns: ['group_id'])]
class Event
{
    /**  @internal */
    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: 'uuid')]
        private Uuid $uuid,
        #[Column(type: 'string(50)')]
        private string $type,
        #[Column(type: 'json', typecast: Json::class)]
        private Json $payload,
        #[Column(type: 'string(25)', typecast: Timestamp::class)]
        private Timestamp $timestamp,
        #[Column(type: 'string', name: 'group_id', nullable: true)]
        private ?string $groupId = null,
        #[Column(type: 'string', nullable: true, typecast: Key::class)]
        private ?Key $project = null,
    ) {}

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

    public function setPayload(Json $payload): void
    {
        $this->payload = $payload;
    }

    public function getTimestamp(): Timestamp
    {
        return $this->timestamp;
    }

    public function updateTimestamp(?Timestamp $timestamp = null): void
    {
        $this->timestamp = $timestamp ?? Timestamp::create();
    }

    public function getProject(): ?Key
    {
        return $this->project;
    }

    public function getGroupId(): ?string
    {
        return $this->groupId;
    }
}
