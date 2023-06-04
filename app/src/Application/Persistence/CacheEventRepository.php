<?php

declare(strict_types=1);

namespace App\Application\Persistence;

use App\Application\Domain\Entity\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Carbon\Carbon;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Spiral\Cache\CacheStorageProviderInterface;

/**
 * @phpstan-type TDocument array{
 *     id: non-empty-string,
 *     type: non-empty-string,
 *     payload: array,
 *     date: positive-int,
 *     projectId: string|null,
 * }
 */
final class CacheEventRepository implements EventRepositoryInterface
{
    private readonly CacheStorage $storage;

    public function __construct(
        CacheStorageProviderInterface $provider,
        $ttl = 60 * 60 * 2
    ) {
        $this->storage = new CacheStorage(
            $provider->storage('events'),
            'events',
            $ttl
        );
    }

    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable
    {
        $events = $this->storage->findAll($scope, $orderBy, $limit, $offset);

        foreach ($events as $document) {
            yield $this->mapDocumentInfoEvent($document);
        }
    }

    public function countAll(array $scope = []): int
    {
        return $this->storage->countAll($scope);
    }

    public function store(Event $event): bool
    {
        return $this->storage->store(
            $event->getUuid(),
            [
                'type' => $event->getType(),
            ],
            [
                'id' => (string)$event->getUuid(),
                'type' => $event->getType(),
                'project_id' => $event->getProjectId(),
                'date' => $event->getDate()->getTimestamp(),
                'payload' => $event->getPayload()->jsonSerialize(),
            ]
        );
    }

    public function deleteAll(array $scope = []): void
    {
        $this->storage->deleteAll($scope);
    }

    public function deleteByPK(string $uuid): bool
    {
        return $this->storage->deleteByPK($uuid);
    }

    /**
     * @param non-empty-string $uuid
     * @throws \Psr\SimpleCache\InvalidArgumentException
     */
    public function findByPK(mixed $uuid): ?Event
    {
        $event = $this->storage->findByPK($uuid);

        if ($event === null) {
            return null;
        }

        return $this->mapDocumentInfoEvent($event);
    }

    public function findOne(array $scope = []): ?Event
    {
        $event = $this->storage->findOne($scope);

        if ($event === null) {
            return null;
        }

        return $this->mapDocumentInfoEvent($event);
    }

    private function mapDocumentInfoEvent(array $document): Event
    {
        return new Event(
            uuid: Uuid::fromString($document['id']),
            type: $document['type'],
            payload: new Json((array)$document['payload']),
            date: Carbon::createFromTimestamp($document['date'])->toDateTimeImmutable(),
            projectId: $document['project_id'],
        );
    }
}
