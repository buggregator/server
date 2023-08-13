<?php

declare(strict_types=1);

namespace Modules\Attachments\Application\Commands;

use App\Application\Commands\StoreAttachment;
use Modules\Attachments\Domain\Attachment;
use Modules\Attachments\Domain\AttachmentRepositoryInterface;
use Spiral\Cqrs\Attribute\CommandHandler;
use Spiral\Storage\BucketInterface;
use Spiral\Storage\Exception\FileOperationException;
use Spiral\Storage\StorageInterface;

final class StoreAttachmentHandler
{
    private readonly BucketInterface $bucket;

    public function __construct(
        private readonly AttachmentRepositoryInterface $repository,
        StorageInterface $storage,
    ) {
        $this->bucket = $storage->bucket('attachments');
    }

    /**
     * @throws FileOperationException
     */
    #[CommandHandler]
    public function handle(StoreAttachment $command): Attachment
    {
        $ext = \pathinfo($command->file->getClientFilename(), PATHINFO_EXTENSION);
        $file = $this->bucket->write(
            $command->uuid . '.' . $ext,
            (string)$command->file->getStream()
        );

        $this->repository->store(
            $attachment = new Attachment(
                $command->uuid,
                $command->parentUuid,
                $command->file->getClientFilename(),
                $file->getPathname(),
                $file->getMimeType(),
                $file->getSize(),
            )
        );

        return $attachment;
    }
}
