<?php

declare(strict_types=1);

namespace Modules\Smtp\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Smtp\Application\Mail\Attachment as MailAttachment;

interface AttachmentStorageInterface
{
    /**
     * @param MailAttachment[] $attachments
     * @return iterable<string, string>
     */
    public function store(Uuid $eventUuid, array $attachments): iterable;

    public function deleteByEvent(Uuid $eventUuid): void;

    /**
     * @return resource Content of the file as a stream
     */
    public function getContent(string $path);
}
