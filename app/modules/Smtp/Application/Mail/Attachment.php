<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mail;

use Ramsey\Uuid\Uuid;

final readonly class Attachment
{
    private string $id;

    public function __construct(
        private ?string $filename,
        private string $content,
        private string $type,
        private ?string $contentId = null,
    ) {
        $this->id = (string) Uuid::uuid4();
    }

    public function getFilename(): string
    {
        return $this->filename;
    }

    public function getContent(): string
    {
        return $this->content;
    }

    public function getType(): string
    {
        return $this->type;
    }

    public function getId(): string
    {
        return $this->id;
    }

    public function getContentId(): ?string
    {
        return $this->contentId;
    }
}
