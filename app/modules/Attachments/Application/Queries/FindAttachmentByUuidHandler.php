<?php

declare(strict_types=1);

namespace Modules\Attachments\Application\Queries;

use App\Application\Commands\FindAttachmentByUuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Attachments\Domain\AttachmentRepositoryInterface;
use Spiral\Cqrs\Attribute\CommandHandler;
use Spiral\Storage\FileInterface;

final class FindAttachmentByUuidHandler
{
    public function __construct(
        private readonly AttachmentRepositoryInterface $repository,
    ) {
    }

    #[CommandHandler]
    public function handle(FindAttachmentByUuid $command): FileInterface
    {
        $attachment = $this->repository->findByPK($command->uuid);
        if (!$attachment) {
            throw new EntityNotFoundException(
                \sprintf('Attachment with uuid %s not found', $command->uuid)
            );
        }

        return $attachment;
    }
}
