<?php

declare(strict_types=1);

namespace Modules\Smtp\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Smtp\Application\Mail\Attachment as MailAttachment;

interface AttachmentStorageInterface
{
    /**
     * @param MailAttachment[] $attachments
     */
    public function store(Uuid $eventUuid, array $attachments): void;

    public function deleteByEvent(Uuid $eventUuid): void;
}
