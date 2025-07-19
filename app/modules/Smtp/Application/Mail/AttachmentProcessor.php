<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mail;

use Modules\Smtp\Application\Mail\Strategy\AttachmentProcessingStrategy;
use ZBateson\MailMimeParser\Message\IMessagePart;

final readonly class AttachmentProcessor
{
    public function __construct(
        private AttachmentProcessingStrategy $strategy,
    ) {}

    /**
     * Processes a message part into an Attachment object
     */
    public function processAttachment(IMessagePart $part): Attachment
    {
        $filename = $this->strategy->generateFilename($part);
        $content = $part->getContent();
        $contentType = $part->getContentType();
        $contentId = $part->getContentId();

        return new Attachment(
            filename: $filename,
            content: $content,
            type: $contentType,
            contentId: $contentId,
        );
    }

    /**
     * Gets metadata about the attachment processing
     */
    public function getMetadata(IMessagePart $part): array
    {
        return $this->strategy->extractMetadata($part);
    }

    /**
     * Determines if the attachment should be stored inline
     */
    public function shouldStoreInline(IMessagePart $part): bool
    {
        return $this->strategy->shouldStoreInline($part);
    }

    /**
     * Gets the current strategy
     */
    public function getStrategy(): AttachmentProcessingStrategy
    {
        return $this->strategy;
    }
}
