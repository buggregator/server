<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Storage;

use Modules\Smtp\Application\Mail\Attachment as MailAttachment;
use Spiral\Storage\StorageInterface;

final readonly class AttachmentStorage
{
    public function __construct(
        private StorageInterface $storage,
    ) {}

    /**
     * @param MailAttachment[] $attachments
     * @return array<non-empty-string, Attachment>
     */
    public function store(string $id, array $attachments): array
    {
        $storedAttachments = [];

        foreach ($attachments as $attachment) {
            $file = $this->storage->write(
                $filename = $id . '/' . $attachment->getFilename(),
                $attachment->getContent(),
            );

            $storedAttachments[$attachment->getId()] = new Attachment(
                name: $attachment->getFilename(),
                uri: $filename,
                size: $file->getSize(),
                mime: $file->getMimeType(),
                id: $attachment->getId(),
            );
        }

        return $storedAttachments;
    }
}
