<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\RepositoryInterface;

interface WebhookRepositoryInterface extends RepositoryInterface
{
    public function store(Webhook $delivery): void;

    /**
     * @param non-empty-string $event
     * @return iterable<Webhook>
     */
    public function findByEvent(string $event): iterable;

    public function getByUuid(Uuid $uuid): Webhook;

    public function findByKey(string $key): ?Webhook;
}
