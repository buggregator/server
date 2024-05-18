<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Smtp;

use App\Application\Domain\ValueObjects\Uuid;
use Database\Factory\AttachmentFactory;
use Modules\Smtp\Domain\Attachment;
use Modules\Smtp\Integration\CycleOrm\AttachmentRepository;
use Spiral\DatabaseSeeder\Database\EntityAssertion;
use Tests\DatabaseTestCase;

final class CycleOrmAttachmentRepositoryTest extends DatabaseTestCase
{
    private EntityAssertion $assertion;
    private AttachmentRepository $repository;

    protected function setUp(): void
    {
        parent::setUp();

        $this->repository = $this->get(AttachmentRepository::class);
        $this->assertion = $this->assertEntity(Attachment::class);
    }

    public function testStore(): void
    {
        $attachment = AttachmentFactory::new()->makeOne();

        $record = $this->assertion->where(['uuid' => $attachment->getUuid()]);

        $record->assertMissing();
        $this->assertion->assertCount(0);

        $this->repository->store($attachment);

        $record->assertExists();
        $this->assertion->assertCount(1);
    }

    public function testFindByUuid(): void
    {
        $eventUuid = Uuid::generate();

        $attachment1 = AttachmentFactory::new()->forEvent($eventUuid)->createOne();
        $attachment2 = AttachmentFactory::new()->forEvent($eventUuid)->createOne();
        $attachment3 = AttachmentFactory::new()->forEvent($eventUuid)->createOne();
        AttachmentFactory::new()->createOne();

        $validUuids = [
            (string) $attachment1->getUuid(),
            (string) $attachment2->getUuid(),
            (string) $attachment3->getUuid(),
        ];

        $this->assertion->assertCount(4);

        $records = $this->repository->findByEvent($eventUuid);

        $this->assertCount(3, $records);

        foreach ($records as $record) {
            $this->assertContains((string) $record->getUuid(), $validUuids);
        }
    }

    public function testDeleteByEvent(): void
    {
        $eventUuid = Uuid::generate();

        AttachmentFactory::new()->forEvent($eventUuid)->times(2)->create();
        $attachment3 = AttachmentFactory::new()->createOne();

        $this->assertion->assertCount(3);

        $this->repository->deleteByEvent($eventUuid);

        $this->assertion->assertCount(1);
        $this->assertion->where(['uuid' => $attachment3->getUuid()])->assertExists();
    }
}
