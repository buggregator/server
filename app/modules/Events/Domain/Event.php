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
    role: Event::ROLE,
    repository: EventRepositoryInterface::class,
    table: Event::TABLE_NAME,
)]
#[Index(columns: ['type'])]
#[Index(columns: ['project'])]
class Event
{
    public const TABLE_NAME = 'events';
    public const ROLE = 'event';

    public const UUID = 'uuid';
    public const TYPE = 'type';
    public const PAYLOAD = 'payload';
    public const TIMESTAMP = 'timestamp';
    public const PROJECT = 'project';

    /**  @internal */
    public function __construct(
        #[Column(type: 'string(36)', name: self::UUID, primary: true, typecast: 'uuid')]
        private Uuid $uuid,
        #[Column(type: 'string(50)', name: self::TYPE)]
        private string $type,
        #[Column(type: 'json', name: self::PAYLOAD, typecast: Json::class)]
        private Json $payload,
        #[Column(type: 'string(25)', name: self::TIMESTAMP, typecast: Timestamp::class)]
        private Timestamp $timestamp,
        #[Column(type: 'string', name: self::PROJECT, nullable: true, typecast: Key::class)]
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

    public function getProject(): ?Key
    {
        return $this->project;
    }
}
