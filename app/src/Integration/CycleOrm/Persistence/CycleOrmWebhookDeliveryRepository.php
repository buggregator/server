<?php

declare(strict_types=1);

namespace App\Integration\CycleOrm\Persistence;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\Select;
use Cycle\ORM\Select\Repository;
use Modules\Webhooks\Domain\Delivery;
use Modules\Webhooks\Domain\DeliveryRepositoryInterface;

/**
 * @template TEntity of Delivery
 * @extends Repository<Delivery>
 */
final class CycleOrmWebhookDeliveryRepository extends Repository implements DeliveryRepositoryInterface
{
    public function __construct(
        private readonly EntityManagerInterface $em,
        Select $select,
    ) {
        parent::__construct($select);
    }

    public function store(Delivery $delivery): void
    {
        $this->em->persist($delivery)->run();
    }

    public function findByWebhook(Uuid $webhookUuid): iterable
    {
        return $this->select()->where('webhookUuid', $webhookUuid)->fetchAll();
    }
}
