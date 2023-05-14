<?php

declare(strict_types=1);

namespace App\Application\Service\Attachments;

use Spiral\Cqrs\Attribute\CommandHandler;
use Spiral\Storage\BucketInterface;
use Spiral\Storage\FileInterface;
use Spiral\Storage\StorageInterface;

final class StoreAttachmentHandler
{
    private readonly BucketInterface $bucket;

    public function __construct(
        StorageInterface $storage,
    ) {
        $this->bucket = $storage->bucket('attachments');
    }

    #[CommandHandler]
    public function handle(\App\Application\Commands\StoreAttachment $command): FileInterface
    {
        $time = microtime();

        return $this->bucket->write(
            $command->type . '/' . $time . '-' . $command->filename,
            $command->content
        );
    }
}
