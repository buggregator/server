<?php

declare(strict_types=1);

namespace Modules\Events\Integration\CycleOrm;

use Cycle\Database\DatabaseInterface;
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
        private readonly DatabaseInterface $db,
        Select $select,
    ) {
        parent::__construct($select);
    }

    public function store(Event $event): bool
    {
        $payload = (string) $event->getPayload();

        try {
            $this->db->insert(Event::TABLE_NAME)->values([
                Event::UUID => (string) $event->getUuid(),
                Event::TYPE => $event->getType(),
                Event::PAYLOAD => $payload,
                Event::TIMESTAMP => (string) $event->getTimestamp(),
                Event::PROJECT => $event->getProject() !== null ? (string) $event->getProject() : null,
                Event::IS_PINNED => $event->isPinned(),
            ])->run();
        } catch (\Throwable) {
            $this->db->update(Event::TABLE_NAME)
                ->where(Event::UUID, (string) $event->getUuid())
                ->values([Event::PAYLOAD => $payload])
                ->run();
        }

        return true;
    }

    public function deleteAll(array $scope = []): void
    {
        $query = $this->db
            ->delete(Event::TABLE_NAME)
            ->where($this->buildScope($scope));

        $query->where(Event::IS_PINNED, false);
        $query->run();
    }

    public function deleteByPK(string $uuid): bool
    {
        return $this->db->delete(Event::TABLE_NAME)
                ->where(Event::UUID, $uuid)
                ->where(Event::IS_PINNED, false)
                ->run() > 0;
    }

    public function pin(string $uuid): bool
    {
        return $this->db->update(Event::TABLE_NAME)
                ->where(Event::UUID, $uuid)
                ->values([Event::IS_PINNED => true])
                ->run() > 0;
    }

    public function unpin(string $uuid): bool
    {
        return $this->db->update(Event::TABLE_NAME)
                ->where(Event::UUID, $uuid)
                ->values([Event::IS_PINNED => false])
                ->run() > 0;
    }

    public function countAll(array $scope = []): int
    {
        return $this->select()
            ->where($this->buildScope($scope))
            ->count();
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
