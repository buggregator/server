<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\RepositoryInterface;

/**
 * @template TEntity of Delivery
 * @extends RepositoryInterface<Delivery>
 */
interface DeliveryRepositoryInterface extends RepositoryInterface
{
    public function store(Delivery $delivery): void;

    /**
     * @return Delivery[]
     */
    public function findByWebhook(Uuid $webhookUuid): iterable;
}
