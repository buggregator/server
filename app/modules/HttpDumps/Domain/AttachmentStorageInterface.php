<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Psr\Http\Message\UploadedFileInterface;

interface AttachmentStorageInterface
{
    /**
     * @param UploadedFileInterface[] $attachments
     * @return array<Attachment>
     */
    public function store(Uuid $eventUuid, array $attachments): array;

    public function deleteByEvent(Uuid $eventUuid): void;

    /**
     * @return resource Content of the file as a stream
     */
    public function getContent(string $path);
}
