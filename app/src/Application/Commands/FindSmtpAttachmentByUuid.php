<?php

declare(strict_types=1);

namespace App\Application\Commands;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Smtp\Domain\Attachment;
use Spiral\Cqrs\QueryInterface;

/**
 * @implements QueryInterface<Attachment>
 *
 * @throws \App\Application\Exception\EntityNotFoundException if attachment with given uuid was not found
 */
final class FindSmtpAttachmentByUuid implements QueryInterface
{
    public function __construct(
        public Uuid $uuid,
    ) {}
}
