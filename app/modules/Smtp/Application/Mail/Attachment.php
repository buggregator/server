<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mail;

use Ramsey\Uuid\Uuid;

final class Attachment
{
    private readonly string $id;

    public function __construct(
        private readonly ?string $filename,
        private readonly string $content,
        private readonly string $type
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
}
