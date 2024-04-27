<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Events\Domain;

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
        $this->assertCount(0, \iterator_to_array($this->repository->findAll()));

        $this->createEvent();

        $this->assertCount(1, \iterator_to_array($this->repository->findAll()));
    }

    public function testDeleteByType(): void
    {
        $this->createEvent(type: 'foo');
        $this->createEvent(type: 'foo');
        $this->createEvent(type: 'bar');

        $this->assertCount(3, \iterator_to_array($this->repository->findAll()));

        $this->repository->deleteAll(['type' => 'foo']);
        $this->assertCount(1, \iterator_to_array($this->repository->findAll()));
    }

    public function testDeleteByUuids(): void
    {
        $this->createEvent(uuid: $uuid1 = $this->randomUuid());
        $this->createEvent(uuid: $uuid2 = $this->randomUuid());
        $this->createEvent(uuid: $uuid3 = $this->randomUuid());

        $this->assertCount(3, \iterator_to_array($this->repository->findAll()));

        $this->repository->deleteAll(['uuid' => [(string)$uuid1, (string)$uuid3]]);
        $this->assertCount(1, \iterator_to_array($this->repository->findAll()));

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

        $this->assertCount(5, \iterator_to_array($this->repository->findAll()));

        $this->repository->deleteAll(['type' => 'foo', 'uuid' => [(string)$uuid1, (string)$uuid3]]);
        $this->assertCount(1, \iterator_to_array($this->repository->findAll()));

        $result = \iterator_to_array($this->repository->findAll());

        $this->assertSame((string)$uuid2, (string)$result[0]->getUuid());
    }
}
