<?php

declare(strict_types=1);

namespace Modules\Smtp\Interfaces\Queries;

use App\Application\Commands\FindSmtpAttachmentByUuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Smtp\Domain\Attachment;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;

final readonly class FindAttachmentByUuidHandler
{
    public function __construct(
        private AttachmentRepositoryInterface $attachments,
    ) {}

    #[QueryHandler]
    public function __invoke(FindSmtpAttachmentByUuid $query): Attachment
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
