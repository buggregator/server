<?php

declare(strict_types=1);

namespace App\Application\Commands;

use App\Application\Domain\ValueObjects\Uuid;
use Psr\Http\Message\UploadedFileInterface;
use Spiral\Cqrs\CommandInterface;

final class StoreAttachment implements CommandInterface
{
    public function __construct(
        public readonly Uuid $uuid,
        public readonly Uuid $parentUuid,
        public readonly UploadedFileInterface $file,
    ) {
    }
}
