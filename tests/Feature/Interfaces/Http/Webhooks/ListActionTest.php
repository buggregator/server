<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Webhooks;

use Database\Factory\WebhookFactory;
use Modules\Webhooks\Domain\Webhook;
use Modules\Webhooks\Interfaces\Http\Resources\WebhookResource;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class ListActionTest extends ControllerTestCase
{
    public function testList(): void
    {
        $webhooks = \array_map(
            static fn(Webhook $webhook) => new WebhookResource($webhook),
            WebhookFactory::new()->times(3)->create(),
        );

        $this->http->listWebhooks()
            ->assertOk()
            ->assertCollectionContainResources($webhooks);
    }
}
