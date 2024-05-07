<?php

declare(strict_types=1);

namespace Database\Factory;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Webhooks\Domain\Delivery;
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
            'webhook_uuid' => Uuid::generate(),
            'payload' => $this->faker->text(),
            'response' => $this->faker->text(),
            'status' => $this->faker->numberBetween(200, 500),
        ];
    }
}
