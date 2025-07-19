<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Smtp\Attachments;

use App\Application\Domain\ValueObjects\Uuid;
use Database\Factory\AttachmentFactory;
use Modules\Smtp\Domain\Attachment;
use Modules\Smtp\Interfaces\Http\Resources\AttachmentResource;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class ListActionTest extends ControllerTestCase
{
    public function testFindAttachments(): void
    {
        $event = $this->createEvent('smtp');
        $attachments = AttachmentFactory::new()->forEvent($event->getUuid())->times(3)->create();
        $missing = AttachmentFactory::new()->createOne();

        $this->http
            ->get('/api/smtp/' . $event->getUuid() . '/attachments')
            ->assertOk()
            ->assertCollectionContainResources(
                \array_map(
                    fn(Attachment $attachment) => new AttachmentResource($attachment),
                    $attachments,
                ),
            )
            ->assertCollectionMissingResources([new AttachmentResource($missing)]);
    }

    public function testFindWithNonExistsEvent(): void
    {
        $eventUuid = Uuid::generate();
        $this->http
            ->get('/api/smtp/' . $eventUuid . '/attachments')
            ->assertNotFound();
    }
}
