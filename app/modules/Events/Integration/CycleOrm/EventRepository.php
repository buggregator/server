<?php

declare(strict_types=1);

namespace Modules\Events\Integration\CycleOrm;

use Cycle\Database\DatabaseInterface;
use Cycle\Database\Injection\Fragment;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\Select;
use Cycle\ORM\Select\Repository;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;

/**
 * @template TEntity of Event
 * @extends Repository<Event>
 */
final class EventRepository extends Repository implements EventRepositoryInterface
{
    public function __construct(
        private readonly EntityManagerInterface $em,
        private readonly DatabaseInterface $db,
        Select $select,
    ) {
        parent::__construct($select);
    }

    public function store(Event $event): bool
    {
        if (($found = $this->findByPK($event->getUuid())) !== null) {
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
        $this->db
            ->delete(Event::TABLE_NAME)
            ->where($this->buildScope($scope))
            ->run();
    }

    public function deleteByPK(string $uuid): bool
    {
        return $this->db->delete(Event::TABLE_NAME)
                ->where(Event::UUID, $uuid)
                ->run() > 0;
    }

    public function countAll(array $scope = []): int
    {
        return $this->select()
            ->where($this->buildScope($scope))
            ->count();
    }

    public function countByType(array $scope = []): array
    {
        return $this->db
            ->select()
            ->from(Event::TABLE_NAME)
            ->columns([
                Event::TYPE,
                new Fragment('COUNT(*) AS cnt'),
            ])
            ->where($this->buildScope($scope))
            ->groupBy(Event::TYPE)
            ->fetchAll();
    }

    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable
    {
        return $this->select()
            ->where($this->buildScope($scope))
            ->orderBy($orderBy)
            ->limit($limit)
            ->offset($offset)
            ->fetchAll();
    }

    private function buildScope(array $scope): array
    {
        $newScope = [];

        foreach ($scope as $key => $value) {
            $newScope[$key] = \is_array($value) ? ['in' => $value] : $value;
        }

        return $newScope;
    }
}
