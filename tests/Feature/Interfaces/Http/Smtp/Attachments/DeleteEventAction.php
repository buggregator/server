<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Smtp\Attachments;

use App\Application\Domain\ValueObjects\Uuid;
use Database\Factory\AttachmentFactory;
use Mockery\MockInterface;
use Modules\Smtp\Domain\Attachment;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;
use Modules\Smtp\Domain\AttachmentStorageInterface;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class DeleteEventAction extends ControllerTestCase
{
    private MockInterface|AttachmentRepositoryInterface $attachements;
    private MockInterface|AttachmentStorageInterface $storage;

    protected function setUp(): void
    {
        parent::setUp();

        $this->attachements = $this->mockContainer(AttachmentRepositoryInterface::class);
        $this->storage = $this->mockContainer(AttachmentStorageInterface::class);
    }

    public function testDeleteEvent(): void
    {
        $event = $this->createEvent('smtp');

        $this->attachements->shouldReceive('deleteByEvent')
            ->once()
            ->with(\Mockery::on(fn(Uuid $uuid) => $uuid->equals($event->getUuid())));

        $this->storage->shouldReceive('deleteByEvent')
            ->once()
            ->with(\Mockery::on(fn(Uuid $uuid) => $uuid->equals($event->getUuid())));

        $this->http
            ->deleteEvent($event->getUuid())
            ->assertSuccessResource();
    }
}
