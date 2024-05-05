<?php

declare(strict_types=1);

namespace Database\Factory;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Events\Domain\Event;
use Modules\Smtp\Domain\Attachment;
use Spiral\DatabaseSeeder\Factory\AbstractFactory;

/**
 * @template TEntity of Attachment
 * @extends AbstractFactory<Attachment>
 */
final class AttachmentFactory extends AbstractFactory
{
    public function makeEntity(array $definition): object
    {
        return new Attachment(
            uuid: $definition['uuid'],
            eventUuid: $definition['event_uuid'],
            name: $definition['name'],
            path: $definition['path'],
            size: $definition['size'],
            mime: $definition['mime'],
            id: $definition['id'],
        );
    }

    public function entity(): string
    {
        return Attachment::class;
    }

    public function definition(): array
    {
        $eventUuid = Uuid::generate();
        return [
            'uuid' => Uuid::generate(),
            'event_uuid' => $eventUuid,
            'name' => $this->faker->word(),
            'path' => $eventUuid . '/' . $this->faker->word() . '.txt',
            'size' => $this->faker->randomNumber(),
            'mime' => $this->faker->randomElement([
                'text/plain',
                'text/html',
                'application/pdf',
                'image/png',
                'image/jpeg',
            ]),
            'id' => \md5((string) Uuid::generate()),
        ];
    }

    public function forEvent(Uuid|Event $uuid): self
    {
        $uuid = $uuid instanceof Event ? $uuid->getUuid() : $uuid;

        return $this->state(fn(\Faker\Generator $faker, array $definition) => [
            'event_uuid' => $uuid,
        ]);
    }
}
