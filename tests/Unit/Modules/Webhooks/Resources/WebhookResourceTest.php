<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Webhooks\Resources;

use Database\Factory\WebhookFactory;
use Modules\Webhooks\Interfaces\Http\Resources\WebhookResource;
use Tests\TestCase;

final class WebhookResourceTest extends TestCase
{
    public function testMap(): void
    {
        $webhook = WebhookFactory::new()->makeOne();

        $resource = new WebhookResource($webhook);

        $this->assertSame(
            [
                'uuid' => (string) $webhook->uuid,
                'event' => $webhook->event,
                'url' => (string) $webhook->url,
                'headers' => $webhook->getHeaders()->jsonSerialize(),
                'verify_ssl' => $webhook->verifySsl,
                'retry_on_failure' => $webhook->retryOnFailure,
            ],
            \json_decode(\json_encode($resource), true),
        );
    }
}
