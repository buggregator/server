<?php

declare(strict_types=1);

namespace Database\Factory;

use App\Application\Domain\Entity\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Database\Factory\Partials\InspectorType;
use Database\Factory\Partials\MonologType;
use Database\Factory\Partials\ProfilerType;
use Database\Factory\Partials\RayType;
use Database\Factory\Partials\SentryType;
use Database\Factory\Partials\SmtpType;
use Database\Factory\Partials\VarDumperType;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\ValueObject\Timestamp;
use Modules\Projects\Domain\ValueObject\Key;
use Spiral\DatabaseSeeder\Factory\AbstractFactory;

/**
 * @template TEntity of Event
 * @extends  AbstractFactory<Event>
 */
final class EventFactory extends AbstractFactory
{
    use SmtpType,
        ProfilerType,
        SentryType,
        MonologType,
        InspectorType,
        VarDumperType,
        RayType;

    public function entity(): string
    {
        return Event::class;
    }

    public function definition(): array
    {
        return [
            'uuid' => Uuid::generate(),
            'type' => $this->faker->randomElement(['sentry', 'monolog', 'var-dump', 'inspector', 'ray', 'profiler']),
            'timestamp' => Timestamp::create(),
            'project' => null,
        ];
    }


    public function makeEntity(array $definition): object
    {
        return new Event(
            uuid: $definition['uuid'],
            type: $definition['type'],
            payload: new Json($this->getPayload($definition['type'])),
            timestamp: $definition['timestamp'],
            project: $definition['project']
                ? ($definition['project'] instanceof Key ? $definition['project'] : Key::create($definition['project']))
                : null,
        );
    }

    public function getPayload(string $type): array
    {
        return match ($type) {
            'sentry' => self::getSentryPayload(),
            'monolog' => self::getMonologPayload(),
            'var-dump' => self::getVarDumperPayload(),
            'inspector' => self::getInspectorPayload(),
            'ray' => self::getRayPayload(),
            'profiler' => self::getProfilerPayload(),
            'smtp' => self::getSmtpPayload(),
            default => ['foo' => 'bar'],
        };
    }
}
