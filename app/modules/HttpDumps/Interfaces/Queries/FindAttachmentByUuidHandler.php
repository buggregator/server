<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Interfaces\Queries;

use App\Application\Commands\FindHttpDumpAttachmentByUuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\HttpDumps\Domain\Attachment;
use Modules\HttpDumps\Domain\AttachmentRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;

final readonly class FindAttachmentByUuidHandler
{
    public function __construct(
        private AttachmentRepositoryInterface $attachments,
    ) {}

    #[QueryHandler]
    public function __invoke(FindHttpDumpAttachmentByUuid $query): Attachment
    {
        $attachment = $this->attachments->findByPK((string) $query->uuid);

        if ($attachment === null) {
            throw new EntityNotFoundException(
                \sprintf('Attachment with given uuid [%s] was not found.', $query->uuid),
            );
        }

        return $attachment;
    }
}
