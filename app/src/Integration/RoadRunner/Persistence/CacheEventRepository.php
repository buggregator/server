<?php

declare(strict_types=1);

namespace App\Integration\RoadRunner\Persistence;

use App\Application\Domain\Entity\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Carbon\Carbon;
use Doctrine\Common\Collections\ArrayCollection;
use Doctrine\Common\Collections\Criteria;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Psr\SimpleCache\CacheInterface;

/**
 * @phpstan-type TDocument array{
 *     id: non-empty-string,
 *     type: non-empty-string,
 *     payload: array,
 *     date: positive-int,
 *     project: string|null,
 * }
 */
final readonly class CacheEventRepository implements EventRepositoryInterface
{
    private const EVENT_IDS_KEY = 'ids';

    public function __construct(
        private CacheInterface $cache,
        private int $ttl = 60 * 60 * 2,
    ) {
    }

    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable
    {
        $events = $this->getFilteredEvents($scope, $orderBy, $limit, $offset);

        foreach ($events as $document) {
            yield $this->mapDocumentIntoEvent($document);
        }
    }

    public function countAll(array $scope = []): int
    {
        $events = $this->getFilteredEvents($scope);

        return \count($events);
    }

    public function store(Event $event): bool
    {
        $id = (string)$event->getUuid();
        $ids = $this->getEventIds();
        $ids[$id] = [
            'uuid' => (string)$event->getUuid(),
            'type' => $event->getType(),
            'project' => $event->getProject(),
            'timestamp' => \microtime(true),
        ];

        $this->cache->set(self::EVENT_IDS_KEY, $ids);

        return $this->cache->set($id, [
            'id' => $id,
            'type' => $event->getType(),
            'project' => $event->getProject(),
            'timestamp' => $event->getTimestamp(),
            'payload' => $event->getPayload()->jsonSerialize(),
        ], Carbon::now()->addSeconds($this->ttl)->diffAsCarbonInterval());
    }

    public function deleteAll(array $scope = []): void
    {
        if ($scope !== []) {
            $ids = $this->getFilteredEventIds($scope);
            $eventIds = $this->getEventIds();
            foreach ($ids as $id) {
                unset($eventIds[$id]);
            }
            $this->cache->set(self::EVENT_IDS_KEY, $eventIds);
        } else {
            $ids = \array_keys($this->getEventIds());
            $this->cache->delete(self::EVENT_IDS_KEY);
        }

        $this->cache->deleteMultiple($ids);
    }

    public function deleteByPK(string $uuid): bool
    {
        $ids = $this->getEventIds();
        if (isset($ids[$uuid])) {
            unset($ids[$uuid]);
            $this->cache->set(self::EVENT_IDS_KEY, $ids);
            $this->cache->delete($uuid);

            return true;
        }

        return false;
    }

    /**
     * @param non-empty-string $uuid
     * @throws \Psr\SimpleCache\InvalidArgumentException
     */
    public function findByPK(mixed $uuid): ?Event
    {
        \assert(\is_string($uuid));

        $event = $this->cache->get($uuid);

        if (\is_array($event)) {
            return $this->mapDocumentIntoEvent($event);
        }

        return null;
    }

    public function findOne(array $scope = []): ?Event
    {
        $events = $this->findAll(scope: $scope, limit: 1);

        foreach ($events as $event) {
            return $this->mapDocumentIntoEvent($event);
        }

        return null;
    }

    /**
     * @return array<non-empty-string, TDocument>
     */
    private function getFilteredEvents(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): array
    {
        $ids = $this->getFilteredEventIds($scope, $orderBy);
        $ids = \array_slice($ids, $offset, $limit);

        $events = \array_filter($this->cache->getMultiple($ids));
        $result = [];

        foreach ($events as $event) {
            $result[$event['id']] = $event;
        }

        unset($events, $criteria, $ids);

        return $result;
    }

    /**
     * @param TDocument $document
     */
    private function mapDocumentIntoEvent(array $document): Event
    {
        return new Event(
            uuid: Uuid::fromString($document['id']),
            type: $document['type'],
            payload: new Json($document['payload']),
            timestamp: $document['timestamp'],
            project: $document['project'],
        );
    }

    private function getFilteredEventIds(array $scope = [], array $orderBy = []): array
    {
        $criteria = (new Criteria())->orderBy($orderBy);
        foreach ($scope as $key => $value) {
            match (true) {
                \is_array($value) => $criteria->orWhere(Criteria::expr()->in($key, $value)),
                null === $value => $criteria->orWhere(Criteria::expr()->isNull($key)),
                default => $criteria->orWhere(Criteria::expr()->eq($key, $value)),
            };
        }

        $ids = (new ArrayCollection($this->getEventIds()))->matching($criteria)->toArray();

        return \array_keys($ids);
    }

    /**
     * @return array<non-empty-string, bool>
     * @throws \Psr\SimpleCache\InvalidArgumentException
     */
    private function getEventIds(): array
    {
        return (array)$this->cache->get(self::EVENT_IDS_KEY, []);
    }
}
