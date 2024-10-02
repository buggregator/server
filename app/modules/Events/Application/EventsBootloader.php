<?php

declare(strict_types=1);

namespace Modules\Events\Application;

use App\Application\Broadcasting\EventMapperRegistryInterface;
use App\Application\Persistence\DriverEnum;
use Cycle\Database\DatabaseInterface;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\ORMInterface;
use Cycle\ORM\Select;
use Modules\Events\Application\Broadcasting\EventsWasClearMapper;
use Modules\Events\Application\Broadcasting\EventWasDeletedMapper;
use Modules\Events\Application\Broadcasting\EventWasReceivedMapper;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Events\Domain\Events\EventsWasClear;
use Modules\Events\Domain\Events\EventWasDeleted;
use Modules\Events\Domain\Events\EventWasReceived;
use Modules\Events\Integration\CycleOrm\EventRepository;
use Modules\Metrics\Application\CollectorRegistryInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Core\FactoryInterface;
use Spiral\RoadRunner\Metrics\Collector;

final class EventsBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            EventRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                DriverEnum $driver,
            ): EventRepositoryInterface => match ($driver) {
                DriverEnum::Database => $factory->make(EventRepository::class),
                default => throw new \Exception('Unsupported database driver'),
            },
            EventRepository::class => static fn(
                ORMInterface $orm,
                EntityManagerInterface $manager,
                DatabaseInterface $db,
            ): EventRepository => new EventRepository(
                em: $manager,
                db: $db,
                select: new Select($orm, Event::class),
            ),
        ];
    }

    public function boot(
        EventMapperRegistryInterface $registry,
        EventWasReceivedMapper $eventWasReceivedMapper,
        CollectorRegistryInterface $collectorRegistry,
    ): void {
        $registry->register(
            event: EventWasDeleted::class,
            mapper: new EventWasDeletedMapper(),
        );

        $registry->register(
            event: EventsWasClear::class,
            mapper: new EventsWasClearMapper(),
        );

        $registry->register(
            event: EventWasReceived::class,
            mapper: $eventWasReceivedMapper,
        );

        $collectorRegistry->register(
            name: 'events',
            collector: Collector::counter()
                ->withHelp('Events counter')
                ->withLabels('type'),
        );
    }
}
