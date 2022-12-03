<?php

declare(strict_types=1);

namespace App\Application\Persistence;

use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\Select;
use Cycle\ORM\Select\Repository;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;

final class CycleOrmEventRepository extends Repository implements EventRepositoryInterface
{
    public function __construct(
        private readonly EntityManagerInterface $entityManager,
        Select $select
    ) {
        parent::__construct($select);
    }

    public function store(Event $event): bool
    {
        $this->entityManager->persist($event);
        $this->entityManager->run();
    }

    public function deleteAll(array $scope = []): void
    {
        $events = $this->findAll($scope);
        foreach ($events as $event) {
            $this->entityManager->delete($event);
        }

        $this->entityManager->run();
    }

    public function deleteByPK(string $uuid): bool
    {
        $event = $this->findByPK($uuid);

        if (!$event) {
            return false;
        }

        $this->entityManager->delete($event);
        $this->entityManager->run();

        return true;
    }

    public function countAll(array $scope = []): int
    {
        return $this->select()->where($scope)->count();
    }

    public function findAll(array $scope = [], array $orderBy = []): iterable
    {
        return $this->select()->where($scope)->orderBy($orderBy)->fetchAll();
    }
}
