<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Application;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\HttpDumps\Domain\Attachment;
use Modules\HttpDumps\Domain\AttachmentFactoryInterface;

final readonly class AttachmentFactory implements AttachmentFactoryInterface
{
    public function create(
        Uuid $eventUuid,
        string $name,
        string $path,
        int $size,
        string $mime,
    ): Attachment {
        return new Attachment(
            uuid: Uuid::generate(),
            eventUuid: $eventUuid,
            name: $name,
            path: $path,
            size: $size,
            mime: $mime,
        );
    }
}
