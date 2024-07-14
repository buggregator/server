<?php

declare(strict_types=1);

namespace Tests;

use App\Application\Persistence\DriverEnum;
use Database\Factory\EventFactory;
use Database\Factory\ProjectFactory;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectRepositoryInterface;
use Modules\Projects\Domain\ValueObject\Key;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;
use Spiral\DatabaseSeeder\Database\Traits\DatabaseAsserts;
use Spiral\DatabaseSeeder\Database\Traits\ShowQueries;
use Spiral\DatabaseSeeder\Database\Traits\Transactions;
use Spiral\DatabaseSeeder\Database\Traits\Helper;

abstract class DatabaseTestCase extends TestCase
{
    use Transactions, Helper, DatabaseAsserts, ShowQueries;

    private DriverEnum $dbDriver;

    protected function setUp(): void
    {
        parent::setUp();

        $this->dbDriver = $this->get(DriverEnum::class);
//        $this->getRefreshStrategy()->enableRefreshAttribute();
    }

    protected function tearDown(): void
    {
        parent::tearDown();

        if ($this->dbDriver === DriverEnum::Database) {
            $this->cleanIdentityMap();
            $this->getCurrentDatabaseDriver()->disconnect();
        }
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
        $project = ProjectFactory::new([
            'key' => Key::create($key),
        ])->makeOne();

        $this->get(ProjectRepositoryInterface::class)->store($project);
        return $project;
    }

    protected function createEvent(string $type = 'fake', ?string $project = null): Event
    {
        $event = EventFactory::new([
            'type' => $type,
            'project' => $project ? Key::create($project) : null,
        ])->makeOne();
        $this->get(EventRepositoryInterface::class)->store($event);

        return $event;
    }

    protected function assertEventExists(Event...$events): self
    {
        $repo = $this->get(EventRepositoryInterface::class);

        foreach ($events as $event) {
            $this->assertNotNull($repo->findByPK($event->getUuid()), 'Event not found in database');
        }

        return $this;
    }

    protected function assertEventMissing(Event ...$events): self
    {
        $repo = $this->get(EventRepositoryInterface::class);
        foreach ($events as $event) {
            $this->assertNull($repo->findByPK($event->getUuid()), 'Event found in database');
        }

        return $this;
    }

    protected function getSmtpAttachmentRepository(): AttachmentRepositoryInterface
    {
        return $this->get(AttachmentRepositoryInterface::class);
    }
}
