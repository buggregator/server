<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mail\Strategy;

use ZBateson\MailMimeParser\Message\IMessagePart;

final readonly class RegularAttachmentStrategy implements AttachmentProcessingStrategy
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
        // Regular attachments don't have content-id or have disposition 'attachment'
        return $part->getContentId() === null ||
               strtolower($part->getContentDisposition()) === 'attachment';
    }

    public function generateFilename(IMessagePart $part): string
    {
        $originalFilename = $part->getFilename();
        $mimeType = $part->getContentType();

        // If we have an original filename, sanitize and use it
        if ($originalFilename !== null && $originalFilename !== '' && $originalFilename !== '0') {
            return $this->sanitizeFilename($originalFilename);
        }

        // Generate a filename based on mime type
        $baseName = 'attachment_' . uniqid();
        $extension = $this->getExtensionFromMimeType($mimeType);

        return $baseName . $extension;
    }

    public function extractMetadata(IMessagePart $part): array
    {
        return [
            'content_id' => $part->getContentId(),
            'is_inline' => false,
            'disposition' => $part->getContentDisposition(),
            'original_filename' => $part->getFilename(),
        ];
    }

    public function shouldStoreInline(IMessagePart $part): bool
    {
        return false;
    }

    public function getPriority(): int
    {
        return 50; // Medium priority for regular attachments
    }

    private function sanitizeFilename(string $filename): string
    {
        // Remove directory traversal attempts
        $filename = basename($filename);

        // Replace problematic characters but preserve more filename characters
        $filename = preg_replace('/[^\w\s\.-]/', '_', $filename);

        // Replace multiple spaces or underscores with single underscore
        $filename = preg_replace('/[\s_]+/', '_', $filename);

        // Remove leading/trailing underscores
        $filename = trim($filename, '_');

        // Ensure we have a reasonable length
        if (strlen($filename) > 255) {
            $pathInfo = pathinfo($filename);
            $extension = isset($pathInfo['extension']) ? '.' . $pathInfo['extension'] : '';
            $baseName = substr($pathInfo['filename'], 0, 255 - strlen($extension));
            $filename = $baseName . $extension;
        }

        // Fallback if filename becomes empty
        if ($filename === '' || $filename === '0') {
            $filename = 'attachment_' . uniqid() . '.bin';
        }

        return $filename;
    }

    private function getExtensionFromMimeType(string $mimeType): string
    {
        $mimeType = strtolower($mimeType);

        return self::MIME_TYPE_EXTENSIONS[$mimeType] ?? '.bin';
    }
}
