<?php

declare(strict_types=1);

namespace Tests\App\Events;

use App\Application\Domain\ValueObjects\Uuid;
use Mockery\MockInterface;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use PHPUnit\Framework\TestCase;

final readonly class EventsMocker
{
    public function __construct(
        private MockInterface&EventRepositoryInterface $events,
    ) {
    }

    public function eventShouldBeFound(Uuid|string $uuid, ?Event $result): self
    {
        $this->events
            ->shouldReceive('findByPK')
            ->with((string)$uuid)
            ->once()
            ->andReturn($result);

        return $this;
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

    public function eventShouldNotBeDeleted(Uuid $uuid): void
    {
        $this->events
            ->shouldNotReceive('deleteByPK')
            ->with((string)$uuid);
    }

    public function eventShouldBeClear(?string $type = null, ?string $project = null, ?array $uuids = null): void
    {
        $args = ['project' => $project];

        if ($type) {
            $args['type'] = $type;
        }

        if ($uuids) {
            $args['uuid'] = $uuids;
        }

        $this->events
            ->shouldReceive('deleteAll')
            ->withArgs(function (array $data) use ($args): bool {
                TestCase::assertSame($args, $data);
                return true;
            })
            ->once();
    }
}
