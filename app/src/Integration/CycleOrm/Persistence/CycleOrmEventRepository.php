<?php

declare(strict_types=1);

namespace App\Integration\CycleOrm\Persistence;

use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\Select;
use Cycle\ORM\Select\Repository;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;

final class CycleOrmEventRepository extends Repository implements EventRepositoryInterface
{
    public function __construct(
        private readonly EntityManagerInterface $em,
        Select $select,
    ) {
        parent::__construct($select);
    }

    public function store(Event $event): bool
    {
        if ($found = $this->findByPK($event->getUuid())) {
            $found->setPayload($event->getPayload());
            $this->em->persist($found);
        } else {
            $this->em->persist($event);
        }

        $this->em->run();

        return true;
    }

    public function deleteAll(array $scope = []): void
    {
        $events = $this->select()->where($scope)->fetchAll();

        foreach ($events as $event) {
            $this->em->delete($event);
        }

        $this->em->run();
    }

    public function deleteByPK(string $uuid): bool
    {
        $event = $this->findByPK($uuid);

        if (!$event) {
            return false;
        }

        $this->em->delete($event);
        $this->em->run();

        return true;
    }

    public function countAll(array $scope = []): int
    {
        return $this->select()->where($scope)->count();
    }

    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable
    {
        return $this->select()->where($scope)->orderBy($orderBy)->limit($limit)->offset($offset)->fetchAll();
    }
}
