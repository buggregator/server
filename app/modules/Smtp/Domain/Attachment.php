<?php

declare(strict_types=1);

namespace Modules\Smtp\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use Cycle\Annotated\Annotation\Table\Index;

#[Entity(
    role: 'smtp_attachment',
    repository: AttachmentRepositoryInterface::class,
    table: 'smtp_attachments',
)]
#[Index(columns: ['event_uuid'])]
class Attachment
{
    /** @internal */
    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: 'uuid')]
        private readonly Uuid $uuid,
        #[Column(type: 'string(36)', typecast: 'uuid')]
        private readonly Uuid $eventUuid,
        #[Column(type: 'string')]
        private readonly string $name,
        #[Column(type: 'string')]
        private readonly string $path,
        #[Column(type: 'integer', default: 0)]
        private readonly int $size,
        #[Column(type: 'string(32)')]
        private readonly string $mime,
        #[Column(type: 'string(255)')]
        private readonly string $id,
    ) {}

    public function getUuid(): Uuid
    {
        return $this->uuid;
    }

    public function getEventUuid(): Uuid
    {
        return $this->eventUuid;
    }

    public function getFilename(): string
    {
        return $this->name;
    }

    public function getPath(): string
    {
        return $this->path;
    }

    public function getSize(): int
    {
        return $this->size;
    }

    public function getMime(): string
    {
        return $this->mime;
    }

    public function getId(): string
    {
        return $this->id;
    }
}
