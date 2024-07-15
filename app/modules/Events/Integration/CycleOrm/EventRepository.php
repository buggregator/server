<?php

declare(strict_types=1);

namespace Modules\Events\Integration\CycleOrm;

use App\Application\Event\StackStrategy;
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
        Select $select,
    ) {
        parent::__construct($select);
    }

    public function store(Event $event, StackStrategy $stackStrategy): bool
    {
        $found = null;
        if ($event->getGroupId() !== null && $stackStrategy === StackStrategy::All) {
            $found = $this->findOne(['group_id' => $event->getGroupId()]);
            if (!$found) {
                $found = $event;
            } else {
                $found->setPayload($event->getPayload());
                $found->updateTimestamp();
            }
        } elseif ($event->getGroupId() !== null && $stackStrategy === StackStrategy::OnlyLatest) {
            $found = $this->findLatest();
            if ($found && $found->getGroupId() === $event->getGroupId()) {
                $found->setPayload($event->getPayload());
                $found->updateTimestamp();
            } else {
                $found = $event;
            }
        }

//        if (!$found && $found = $this->findByPK($event->getUuid())) {
//            $found->setPayload($event->getPayload());
//            $found->updateTimestamp();
//        }

        if ($found) {
            $this->em->persist($found);
        } else {
            $this->em->persist($event);
        }

        $this->em->run();

        return true;
    }

    public function deleteAll(array $scope = []): void
    {
        $events = $this->select()
            ->where($this->buildScope($scope))
            ->fetchAll();

        $batchSize = 100;

        $batch = 0;
        foreach ($events as $event) {
            $this->em->delete($event);

            if (++$batch % $batchSize === 0) {
                $this->em->run();
                $batch = 0;
            }
        }

        $this->em->run();
    }

    public function deleteByPK(string $uuid): bool
    {
        $event = $this->findByPK($uuid);

        if ($event === null) {
            return false;
        }

        $this->em->delete($event)->run();

        return true;
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

    private function findLatest(): ?Event
    {
        return $this->select()
            ->orderBy(['timestamp' => 'DESC'])
            ->fetchOne();
    }
}
