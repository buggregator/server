<?php

declare(strict_types=1);

namespace App\Integration\CycleOrm\Persistence;

use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\RepositoryInterface;
use Cycle\ORM\Select;
use Cycle\ORM\Select\Repository;
use Modules\Webhooks\Domain\Webhook;
use Modules\Webhooks\Domain\WebhookRepositoryInterface;

/**
 * @template TEntity of Webhook
 * @extends RepositoryInterface<Webhook>
 */
final class CycleOrmWebhookRepository extends Repository implements WebhookRepositoryInterface
{
    public function __construct(
        private readonly EntityManagerInterface $em,
        Select $select,
    ) {
        parent::__construct($select);
    }

    public function store(Webhook $delivery): void
    {
        $this->em->persist($delivery)->run();
    }

    public function findByEvent(string $event): iterable
    {
        return $this->select()->where('event', $event)->fetchAll();
    }

    public function getByUuid(Uuid $uuid): Webhook
    {
        return $this->findByPK((string) $uuid) ?? throw new EntityNotFoundException(
            \sprintf('Webhook with uuid %s not found', $uuid),
        );
    }

    public function findByKey(string $key): ?Webhook
    {
        return $this->findOne(['key' => $key]);
    }
}
