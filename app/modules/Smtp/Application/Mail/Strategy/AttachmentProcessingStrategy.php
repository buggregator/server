<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mail\Strategy;

use ZBateson\MailMimeParser\Message\IMessagePart;

interface AttachmentProcessingStrategy
{
    /**
     * Determines if this strategy can handle the given message part
     */
    public function canHandle(IMessagePart $part): bool;

    /**
     * Generates a safe filename for the attachment
     */
    public function generateFilename(IMessagePart $part): string;

    /**
     * Extracts metadata from the message part
     */
    public function extractMetadata(IMessagePart $part): array;

    /**
     * Determines if the attachment should be stored inline
     */
    public function shouldStoreInline(IMessagePart $part): bool;

    /**
     * Gets the priority of this strategy (higher number = higher priority)
     */
    public function getPriority(): int;
}
