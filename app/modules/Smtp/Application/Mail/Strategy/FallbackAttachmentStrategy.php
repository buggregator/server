<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mail\Strategy;

use ZBateson\MailMimeParser\Message\IMessagePart;

final readonly class FallbackAttachmentStrategy implements AttachmentProcessingStrategy
{
    public function canHandle(IMessagePart $part): bool
    {
        // Fallback strategy handles everything
        return true;
    }

    public function generateFilename(IMessagePart $part): string
    {
        $originalFilename = $part->getFilename();
        $mimeType = $part->getContentType();
        $contentId = $part->getContentId();

        // Try original filename first
        if ($originalFilename !== null && $originalFilename !== '' && $originalFilename !== '0') {
            return $this->sanitizeFilename($originalFilename);
        }

        // Try content-id if available
        if ($contentId !== null && $contentId !== '' && $contentId !== '0') {
            $safeName = $this->sanitizeContentId($contentId);
            $extension = $this->getExtensionFromMimeType($mimeType);
            return $safeName . $extension;
        }

        // Last resort: generate unique filename
        $baseName = 'unknown_attachment_' . uniqid();
        $extension = $this->getExtensionFromMimeType($mimeType);

        return $baseName . $extension;
    }

    public function extractMetadata(IMessagePart $part): array
    {
        return [
            'content_id' => $part->getContentId(),
            'is_inline' => $part->getContentDisposition() === 'inline',
            'disposition' => $part->getContentDisposition(),
            'original_filename' => $part->getFilename(),
            'fallback_used' => true,
        ];
    }

    public function shouldStoreInline(IMessagePart $part): bool
    {
        return $part->getContentDisposition() === 'inline';
    }

    public function getPriority(): int
    {
        return 1; // Lowest priority - fallback only
    }

    private function sanitizeFilename(string $filename): string
    {
        // Remove directory traversal attempts
        $filename = basename($filename);

        // Replace problematic characters
        $filename = preg_replace('/[^\w\s\.-]/', '_', $filename);

        // Replace multiple spaces or underscores with single underscore
        $filename = preg_replace('/[\s_]+/', '_', $filename);

        // Remove leading/trailing underscores
        $filename = trim($filename, '_');

        // Ensure we have a reasonable length
        if (strlen($filename) > 255) {
            $filename = substr($filename, 0, 255);
        }

        // Fallback if filename becomes empty
        if ($filename === '' || $filename === '0') {
            $filename = 'fallback_' . uniqid() . '.bin';
        }

        return $filename;
    }

    private function sanitizeContentId(string $contentId): string
    {
        // Remove angle brackets if present
        $contentId = trim($contentId, '<>');

        // Replace problematic characters with underscores
        $safeName = preg_replace('/[^a-zA-Z0-9._-]/', '_', $contentId);

        // Remove multiple consecutive underscores
        $safeName = preg_replace('/_+/', '_', $safeName);

        // Trim underscores from start and end
        $safeName = trim($safeName, '_');

        // If empty or too short, generate a fallback
        if ($safeName === '' || $safeName === '0' || strlen($safeName) < 3) {
            $safeName = 'cid_' . uniqid();
        }

        return $safeName;
    }

    private function getExtensionFromMimeType(string $mimeType): string
    {
        $mimeType = strtolower($mimeType);

        $extensions = [
            'image/jpeg' => '.jpg',
            'image/jpg' => '.jpg',
            'image/png' => '.png',
            'image/gif' => '.gif',
            'image/svg+xml' => '.svg',
            'application/pdf' => '.pdf',
            'text/plain' => '.txt',
            'text/html' => '.html',
            'application/zip' => '.zip',
            'application/json' => '.json',
            'application/xml' => '.xml',
        ];

        return $extensions[$mimeType] ?? '.bin';
    }
}
