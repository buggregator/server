<?php

declare(strict_types=1);

namespace App\Integration\MongoDb\Mappers;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Smtp\Domain\Attachment;
use MongoDB\Model\BSONDocument;

final readonly class SmtpAttachmentMapper
{
    public const EVENT_UUID = 'event_uuid';

    public function toDocument(Attachment $attachment): array
    {
        return [
            '_id' => (string) $attachment->getUuid(),
            self::EVENT_UUID => (string) $attachment->getEventUuid(),
            'name' => $attachment->getFilename(),
            'path' => $attachment->getPath(),
            'size' => $attachment->getSize(),
            'mime' => $attachment->getMime(),
            'id' => $attachment->getId(),
        ];
    }

    public function toAttachment(BSONDocument $document): Attachment
    {
        /** @psalm-suppress InternalMethod */
        return new Attachment(
            uuid: new Uuid($document['_id']),
            eventUuid: new Uuid($document[self::EVENT_UUID]),
            name: $document['name'],
            path: $document['path'],
            size: (int) $document['size'],
            mime: $document['mime'],
            id: $document['id'],
        );
    }
}
