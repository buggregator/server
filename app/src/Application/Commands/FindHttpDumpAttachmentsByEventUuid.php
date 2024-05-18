<?php

declare(strict_types=1);

namespace App\Application\Commands;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\HttpDumps\Domain\Attachment;
use Spiral\Cqrs\QueryInterface;

/**
 * @implements QueryInterface<Attachment[]>
 */
final readonly class FindHttpDumpAttachmentsByEventUuid implements QueryInterface
{
    public function __construct(public Uuid $uuid) {}
}
