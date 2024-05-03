<?php

declare(strict_types=1);

namespace Modules\Smtp\Interfaces\Queries;

use App\Application\Commands\FindAttachmentsByEventUuid;
use App\Application\Commands\FindEventByUuid;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;
use Spiral\Cqrs\QueryBusInterface;

final readonly class FindAttachmentsByEventHandler
{
    public function __construct(
        private AttachmentRepositoryInterface $attachments,
        private QueryBusInterface $bus,
    ) {}

    #[QueryHandler]
    public function __invoke(FindAttachmentsByEventUuid $query): iterable
    {
        $this->bus->ask(new FindEventByUuid($query->uuid));

        return $this->attachments->findByEvent($query->uuid);
    }
}
