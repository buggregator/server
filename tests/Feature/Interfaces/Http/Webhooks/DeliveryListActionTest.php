<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Webhooks;

use Database\Factory\WebhookDeliveryFactory;
use Database\Factory\WebhookFactory;
use Modules\Webhooks\Domain\Delivery;
use Modules\Webhooks\Interfaces\Http\Resources\DeliveryResource;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class DeliveryListActionTest extends ControllerTestCase
{
    public function testList(): void
    {
        $webhook = WebhookFactory::new()->createOne();
        $deliveries = WebhookDeliveryFactory::new()
            ->forWebhook($webhook)
            ->times(3)
            ->create();

        $missingDeliveries = WebhookDeliveryFactory::new()
            ->times(2)
            ->create();

        $deliveries = \array_map(
            static fn(Delivery $delivery) => new DeliveryResource($delivery),
            $deliveries,
        );

        $missingDeliveries = \array_map(
            static fn(Delivery $delivery) => new DeliveryResource($delivery),
            $missingDeliveries,
        );

        $this->http->listWebhookDeliveries($webhook->uuid)
            ->assertOk()
            ->assertCollectionContainResources($deliveries)
            ->assertCollectionMissingResources($missingDeliveries);
    }
}
