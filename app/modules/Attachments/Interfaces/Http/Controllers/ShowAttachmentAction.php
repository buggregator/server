<?php

declare(strict_types=1);

namespace Modules\Attachments\Interfaces\Http\Controllers;

use App\Application\Commands\FindAttachmentByUuid;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Attachments\Domain\Attachment;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Router\Annotation\Route;
use Spiral\Storage\BucketInterface;
use Spiral\Storage\StorageInterface;

final class ShowAttachmentAction
{
    private readonly BucketInterface $bucket;

    public function __construct(
        StorageInterface $storage,
    ) {
        $this->bucket = $storage->bucket('attachments');
    }

    #[Route(route: 'attachment/<uuid>', name: 'attachment.show', group: 'api')]
    public function __invoke(
        QueryBusInterface $bus,
        Uuid $uuid,
    ): AttachmentResource {
        try {
            /** @var Attachment $attachment */
            $attachment = $bus->ask(new FindAttachmentByUuid($uuid));
            return new AttachmentResource($attachment);

        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }
    }
}
