<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mail\Strategy;

use ZBateson\MailMimeParser\Message\IMessagePart;

final readonly class InlineAttachmentStrategy implements AttachmentProcessingStrategy
{
    private const MIME_TYPE_EXTENSIONS = [
        'image/jpeg' => '.jpg',
        'image/jpg' => '.jpg',
        'image/png' => '.png',
        'image/gif' => '.gif',
        'image/svg+xml' => '.svg',
        'image/webp' => '.webp',
        'image/bmp' => '.bmp',
        'image/tiff' => '.tiff',
        'application/pdf' => '.pdf',
        'text/plain' => '.txt',
        'text/html' => '.html',
        'text/css' => '.css',
        'application/javascript' => '.js',
        'application/json' => '.json',
        'application/xml' => '.xml',
        'application/zip' => '.zip',
        'application/x-zip-compressed' => '.zip',
        'application/msword' => '.doc',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document' => '.docx',
        'application/vnd.ms-excel' => '.xls',
        'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' => '.xlsx',
    ];

    public function canHandle(IMessagePart $part): bool
    {
        return $part->getContentId() !== null;
    }

    public function generateFilename(IMessagePart $part): string
    {
        $contentId = $part->getContentId();
        $mimeType = $part->getContentType();
        $originalFilename = $part->getFilename();

        // If we have an original filename, use it
        if ($originalFilename !== null && $originalFilename !== '' && $originalFilename !== '0') {
            return $this->sanitizeFilename($originalFilename);
        }

        // Generate filename from content-id
        $safeName = $this->sanitizeContentId($contentId);
        $extension = $this->getExtensionFromMimeType($mimeType);

        return $safeName . $extension;
    }

    public function extractMetadata(IMessagePart $part): array
    {
        return [
            'content_id' => $part->getContentId(),
            'is_inline' => true,
            'disposition' => $part->getContentDisposition(),
            'original_filename' => $part->getFilename(),
        ];
    }

    public function shouldStoreInline(IMessagePart $part): bool
    {
        return true;
    }

    public function getPriority(): int
    {
        return 100; // High priority for inline attachments
    }

    private function sanitizeContentId(string $contentId): string
    {
        // Remove angle brackets if present
        $contentId = \trim($contentId, '<>');

        // Replace problematic characters with underscores
        $safeName = \preg_replace('/[^a-zA-Z0-9._-]/', '_', $contentId);

        // Remove multiple consecutive underscores
        $safeName = \preg_replace('/_+/', '_', $safeName);

        // Trim underscores from start and end
        $safeName = \trim($safeName, '_');

        // If empty or too short, generate a fallback
        if ($safeName === '' || $safeName === '0' || strlen($safeName) < 3) {
            $safeName = 'inline_' . uniqid();
        }

        return $safeName;
    }

    private function sanitizeFilename(string $filename): string
    {
        // Remove directory traversal attempts
        $filename = basename($filename);

        // Replace problematic characters
        $filename = preg_replace('/[^\w\.-]/', '_', $filename);

        // Remove multiple consecutive underscores
        $filename = preg_replace('/_+/', '_', $filename);

        // Ensure we have a reasonable length
        if (strlen($filename) > 255) {
            $filename = substr($filename, 0, 255);
        }

        return $filename;
    }

    private function getExtensionFromMimeType(string $mimeType): string
    {
        $mimeType = strtolower($mimeType);

        return self::MIME_TYPE_EXTENSIONS[$mimeType] ?? '.bin';
    }
}
