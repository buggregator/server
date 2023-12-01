<?php

declare(strict_types=1);

namespace Tests\App\Events;

use App\Application\Domain\ValueObjects\Uuid;
use Mockery\MockInterface;
use Modules\Events\Domain\EventRepositoryInterface;

final class EventsMocker
{
    public function __construct(
        private readonly MockInterface&EventRepositoryInterface $events,
    ) {
    }

    public function shouldRequestEventByUuid(Uuid $uuid): EventExpectation
    {
        return new EventExpectation(
            $this->events
                ->shouldReceive('findByPK')
                ->with((string)$uuid)
                ->once(),
        );
    }

    public function eventShouldBeDeleted(Uuid $uuid, bool $status = true): void
    {
        $this->events
            ->shouldReceive('deleteByPK')
            ->with((string)$uuid)
            ->once()
            ->andReturn($status);
    }

    public function eventShouldBeClear(?string $type = null): void
    {
        $this->events
            ->shouldReceive('deleteAll')
            ->with($type ? ['type' => $type] : [])
            ->once();
    }
}
