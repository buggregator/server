<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Query;

use App\Application\Commands\FindWebhooks;
use Modules\Webhooks\Domain\WebhookRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;

final readonly class FindWebhooksHandler
{
    public function __construct(
        private WebhookRepositoryInterface $webhooks,
    ) {}

    #[QueryHandler]
    public function __invoke(FindWebhooks $query): iterable
    {
        return $this->webhooks->findAll();
    }
}
