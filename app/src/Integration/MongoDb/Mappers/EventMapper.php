<?php

declare(strict_types=1);

namespace App\Integration\MongoDb\Mappers;

use App\Application\Domain\Entity\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\ValueObject\Timestamp;
use Modules\Projects\Domain\ValueObject\Key;
use MongoDB\Model\BSONDocument;

final readonly class EventMapper
{
    public function toDocument(Event $event): array
    {
        return [
            '_id' => (string) $event->getUuid(),
            'type' => $event->getType(),
            'project' => $event->getProject() ? (string) $event->getProject() : null,
            'timestamp' => (string) $event->getTimestamp(),
            'payload' => $event->getPayload()->jsonSerialize(),
        ];
    }

    public function toEvent(BSONDocument $document): Event
    {
        /** @psalm-suppress InternalMethod */
        return new Event(
            uuid: Uuid::fromString($document['_id']),
            type: $document['type'],
            payload: new Json((array) $document['payload']),
            timestamp: new Timestamp($document['timestamp']),
            project: $document['project'] ? new Key($document['project']) : null,
        );
    }
}
