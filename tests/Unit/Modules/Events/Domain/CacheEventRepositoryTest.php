<?php

declare(strict_types=1);

namespace Modules\Events\Domain;

use App\Application\Domain\Entity\Json;
use App\Application\Domain\ValueObjects\Uuid;
use App\Integration\RoadRunner\Persistence\CacheEventRepository;
use Tests\TestCase;

final class CacheEventRepositoryTest extends TestCase
{
    private CacheEventRepository $repository;

    protected function setUp(): void
    {
        parent::setUp();

        $this->repository = $this->get(CacheEventRepository::class);
    }

    public function testStoreEvent(): void
    {
        $this->assertCount(0, $this->repository->findAll());

        $this->createEvent();

        $this->assertCount(1, $this->repository->findAll());
    }

    public function testDeleteByType(): void
    {
        $this->createEvent(type: 'foo');
        $this->createEvent(type: 'foo');
        $this->createEvent(type: 'bar');

        $this->assertCount(3, $this->repository->findAll());

        $this->repository->deleteAll(['type' => 'foo']);
        $this->assertCount(1, $this->repository->findAll());
    }

    public function testDeleteByUuids(): void
    {
        $this->createEvent(uuid: $uuid1 = $this->randomUuid());
        $this->createEvent(uuid: $uuid2 = $this->randomUuid());
        $this->createEvent(uuid: $uuid3 = $this->randomUuid());

        $this->assertCount(3, $this->repository->findAll());

        $this->repository->deleteAll(['uuid' => [(string)$uuid1, (string)$uuid3]]);
        $this->assertCount(1, $this->repository->findAll());

        $result = \iterator_to_array($this->repository->findAll());

        $this->assertSame((string)$uuid2, (string)$result[0]->getUuid());
    }

    public function testDeleteByTypeAndUuids(): void
    {
        $this->createEvent(type: 'foo');
        $this->createEvent(type: 'foo');
        $this->createEvent(type: 'bar', uuid: $uuid1 = $this->randomUuid());
        $this->createEvent(uuid: $uuid2 = $this->randomUuid());
        $this->createEvent(uuid: $uuid3 = $this->randomUuid());

        $this->assertCount(5, $this->repository->findAll());

        $this->repository->deleteAll(['type' => 'foo', 'uuid' => [(string)$uuid1, (string)$uuid3]]);
        $this->assertCount(1, $this->repository->findAll());

        $result = \iterator_to_array($this->repository->findAll());

        $this->assertSame((string)$uuid2, (string)$result[0]->getUuid());
    }

    private function createEvent(string $type = 'test', ?Uuid $uuid = null): Event
    {
        $this->repository->store(
            $event = new Event(
                uuid: $uuid ?? $this->randomUuid(),
                type: $type,
                payload: new Json([]),
                timestamp: \microtime(true),
                projectId: null,
            ),
        );

        return $event;
    }
}
