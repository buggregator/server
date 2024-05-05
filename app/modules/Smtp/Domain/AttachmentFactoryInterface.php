<?php

declare(strict_types=1);

namespace Modules\Smtp\Domain;

use App\Application\Domain\ValueObjects\Uuid;

interface AttachmentFactoryInterface
{
    public function create(
        Uuid $eventUuid,
        string $name,
        string $path,
        int $size,
        string $mime,
        string $id,
    ): Attachment;
}
