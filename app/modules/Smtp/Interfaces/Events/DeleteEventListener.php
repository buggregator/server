<?php

declare(strict_types=1);

namespace Modules\Smtp\Interfaces\Events;

use Modules\Events\Domain\Events\EventWasDeleted;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;
use Modules\Smtp\Domain\AttachmentStorageInterface;
use Spiral\Events\Attribute\Listener;

final readonly class DeleteEventListener
{
    public function __construct(
        private AttachmentStorageInterface $service,
        private AttachmentRepositoryInterface $attachments,
    ) {}

    #[Listener]
    public function __invoke(EventWasDeleted $event): void
    {
        $this->attachments->deleteByEvent($event->uuid);
        $this->service->deleteByEvent($event->uuid);
    }
}
