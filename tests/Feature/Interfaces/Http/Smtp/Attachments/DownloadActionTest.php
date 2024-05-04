<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Smtp\Attachments;

use App\Application\Domain\ValueObjects\Uuid;
use App\Application\HTTP\Response\JsonResource;
use Database\Factory\AttachmentFactory;
use Spiral\Testing\Storage\FakeBucket;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class DownloadActionTest extends ControllerTestCase
{
    public function testDownloadContent(): void
    {
        $storage = $this->fakeStorage();

        $event = $this->createEvent('smtp');
        $attachment = AttachmentFactory::new([
            'name' => 'file.txt',
        ])->forEvent($event)->createOne();

        /** @var FakeBucket $bucket */
        $bucket = $storage->bucket('attachments');

        $bucket->write($attachment->getPath(), $content = 'Downloaded content');

        $this->http->get('/api/smtp/' . $event->getUuid() . '/attachments/' . $attachment->getUuid())
            ->assertOk()
            ->assertBodySame($content)
            ->assertHasHeader('Content-Type', 'application/octet-stream')
            ->assertHasHeader('Content-Length', (string) \strlen($content))
            ->assertHasHeader('Content-Disposition', 'attachment; filename="file.txt"');
    }


    public function testDownloadFromNonExistsEvent(): void
    {
        $eventUuid = Uuid::generate();
        $attachmentUuid = Uuid::generate();

        $this->http->get('/api/smtp/' . $eventUuid . '/attachments/' . $attachmentUuid)
            ->assertNotFound()
            ->assertResource(
                new JsonResource([
                    'message' => 'Event with given uuid [' . $eventUuid . '] was not found.',
                    'code' => 404,
                ]),
            );
    }

    public function testDownloadFromNonExistsAttachment(): void
    {
        $event = $this->createEvent('smtp');
        $attachmentUuid = Uuid::generate();

        $this->http->get('/api/smtp/' . $event->getUuid() . '/attachments/' . $attachmentUuid)
            ->assertNotFound()
            ->assertResource(
                new JsonResource([
                    'message' => 'Attachment with given uuid [' . $attachmentUuid . '] was not found.',
                    'code' => 404,
                ]),
            );
    }

    public function testDownloadAttachmentNotBelongToEvent(): void
    {
        $event = $this->createEvent('smtp');
        $attachment = AttachmentFactory::new()->createOne();

        $this->http->get('/api/smtp/' . $event->getUuid() . '/attachments/' . $attachment->getUuid())
            ->assertForbidden()
            ->assertResource(
                new JsonResource([
                    'message' => 'Access denied.',
                    'code' => 403,
                ]),
            );
    }
}
