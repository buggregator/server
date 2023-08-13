<?php

declare(strict_types=1);

namespace Modules\Attachments\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use Modules\Events\Domain\EventRepositoryInterface;

#[Entity(
    repository: EventRepositoryInterface::class
)]
final class Attachment
{
    /**  @internal */
    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: 'uuid')]
        private Uuid $uuid,

        #[Column(type: 'string(36)', typecast: 'uuid')]
        private Uuid $parentUuid,

        #[Column(type: 'string')]
        private string $filename,

        #[Column(type: 'string')]
        private string $path,

        #[Column(type: 'string')]
        private string $mimeType,

        #[Column(type: 'integer')]
        private int $size,
    ) {
    }

    public function getUuid(): Uuid
    {
        return $this->uuid;
    }

    public function getParentUuid(): Uuid
    {
        return $this->parentUuid;
    }

    public function getFilename(): string
    {
        return $this->filename;
    }

    public function getPath(): string
    {
        return $this->path;
    }

    public function getMimeType(): string
    {
        return $this->mimeType;
    }

    /**
     * @return positive-int
     */
    public function getSize(): int
    {
        return $this->size;
    }
}
