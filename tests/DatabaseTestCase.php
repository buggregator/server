<?php

declare(strict_types=1);

namespace Tests;

use Database\Factory\EventFactory;
use Database\Factory\ProjectFactory;
use Modules\Events\Domain\Event;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ValueObject\Key;
use Spiral\DatabaseSeeder\Database\Traits\DatabaseAsserts;
use Spiral\DatabaseSeeder\Database\Traits\DatabaseMigrations;
use Spiral\DatabaseSeeder\Database\Traits\ShowQueries;
use Spiral\DatabaseSeeder\Database\Traits\Transactions;
use Spiral\DatabaseSeeder\Database\Traits\Helper;
use Spiral\DatabaseSeeder\Database\Traits\RefreshDatabase;

abstract class DatabaseTestCase extends TestCase
{
    use DatabaseMigrations, RefreshDatabase, Transactions, Helper, DatabaseAsserts, ShowQueries;

    protected function setUp(): void
    {
        parent::setUp();

        $this->getRefreshStrategy()->enableRefreshAttribute();
    }

    protected function tearDown(): void
    {
        parent::tearDown();

        $this->cleanIdentityMap();
        $this->getCurrentDatabaseDriver()->disconnect();
    }

    public function persist(object ...$entity): void
    {
        $em = $this->getEntityManager();
        foreach ($entity as $e) {
            $em->persist($e);
        }
        $em->run();
    }

    /**
     * @template T of object
     * @param T $entity
     * @return T
     */
    public function refreshEntity(object $entity, string $pkField = 'uuid'): ?object
    {
        return $this->getRepositoryFor($entity)->findByPK($entity->{$pkField});
    }

    protected function createProject(string $key = 'default'): Project
    {
        return ProjectFactory::new([
            'key' => Key::create($key),
        ])->createOne();
    }

    protected function createEvent(string $type = 'fake', ?string $project = null): Event
    {
        return EventFactory::new([
            'type' => $type,
            'project' => $project ? Key::create($project) : null,
        ])->createOne();
    }

    protected function assertEventExists(Event...$events): self
    {
        foreach ($events as $event) {
            $this->assertEntity($event)->where([
                'uuid' => $event->getUuid(),
            ])->assertExists();
        }

        return $this;
    }

    protected function assertEventMissing(Event ...$events): self
    {
        foreach ($events as $event) {
            $this->assertEntity($event)->where([
                'uuid' => $event->getUuid(),
            ])->assertMissing();
        }

        return $this;
    }
}
