<?php

declare(strict_types=1);

namespace App\Application\Commands;

use App\Application\Exception\EntityNotFoundException;
use App\Application\Domain\ValueObjects\Uuid;
use Modules\HttpDumps\Domain\Attachment;
use Spiral\Cqrs\QueryInterface;

/**
 * @implements QueryInterface<Attachment>
 *
 * @throws EntityNotFoundException if attachment with given uuid was not found
 */
final class FindHttpDumpAttachmentByUuid implements QueryInterface
{
    public function __construct(
        public Uuid $uuid,
    ) {}
}
