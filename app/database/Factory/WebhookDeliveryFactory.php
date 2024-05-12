<?php

declare(strict_types=1);

namespace Database\Factory;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Webhooks\Domain\Delivery;
use Modules\Webhooks\Domain\Webhook;
use Spiral\DatabaseSeeder\Factory\AbstractFactory;

/**
 * @template TEntity of Delivery
 * @extends AbstractFactory<Delivery>
 */
final class WebhookDeliveryFactory extends AbstractFactory
{
    public function makeEntity(array $definition): object
    {
        $uuid = $definition['uuid'] ?? Uuid::generate();

        return new Delivery(
            uuid: $uuid,
            webhookUuid: $definition['webhook_uuid'],
            payload: $definition['payload'],
            response: $definition['response'],
            status: $definition['status'],
        );
    }

    public function entity(): string
    {
        return Delivery::class;
    }

    public function definition(): array
    {
        return [
            'webhook_uuid' => static fn() => WebhookFactory::new()->createOne()->uuid,
            'payload' => $this->faker->text(),
            'response' => $this->faker->text(),
            'status' => $this->faker->numberBetween(200, 500),
        ];
    }

    public function forWebhook(Webhook|Uuid $uuid): self
    {
        return $this->state(static fn() => [
            'webhook_uuid' => $uuid instanceof Uuid ? $uuid : $uuid->uuid,
        ]);
    }
}
