<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Events\Domain;

use Modules\Events\Domain\EventRepositoryInterface;
use Tests\DatabaseTestCase;

final class CacheEventRepositoryTest extends DatabaseTestCase
{
    private EventRepositoryInterface $repository;

    protected function setUp(): void
    {
        parent::setUp();

        $this->repository = $this->get(EventRepositoryInterface::class);
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
        $event1 = $this->createEvent();
        $event2 = $this->createEvent();
        $event3 = $this->createEvent();

        $this->assertCount(3, \iterator_to_array($this->repository->findAll()));

        $this->repository->deleteAll([
            'uuid' => [
                (string) $event1->getUuid(),
                (string) $event3->getUuid(),
            ],
        ]);
        $this->assertCount(1, \iterator_to_array($this->repository->findAll()));

        $result = \iterator_to_array($this->repository->findAll());

        $this->assertSame((string) $event2->getUuid(), (string) $result[0]->getUuid());
    }

    public function testDeleteByTypeAndUuids(): void
    {
        $event1 = $this->createEvent(type: 'foo');
        $event2 = $this->createEvent(type: 'foo');
        $event3 = $this->createEvent(type: 'bar');
        $event4 = $this->createEvent(type: 'foo');
        $event5 = $this->createEvent(type: 'foo');

        $this->assertCount(5, \iterator_to_array($this->repository->findAll()));

        $this->repository->deleteAll([
            'type' => 'foo',
            'uuid' => [
                (string) $event3->getUuid(),
                (string) $event5->getUuid(),
                (string) $event4->getUuid(),
            ],
        ]);

        $this->assertCount(3, \iterator_to_array($this->repository->findAll()));

        $result = \iterator_to_array($this->repository->findAll());

        $this->assertSame((string) $event1->getUuid(), (string) $result[0]->getUuid());
        $this->assertSame((string) $event2->getUuid(), (string) $result[1]->getUuid());
        $this->assertSame((string) $event3->getUuid(), (string) $result[2]->getUuid());
    }
}
