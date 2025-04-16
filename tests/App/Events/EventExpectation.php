<?php

declare(strict_types=1);

namespace Tests\App\Events;

use App\Application\Exception\EntityNotFoundException;
use Mockery\CompositeExpectation;
use Modules\Events\Domain\Event;

final class EventExpectation
{
    public function __construct(
        private CompositeExpectation $expectation,
    ) {}

    public function andReturnEvent(Event $event): void
    {
        $this->expectation->andReturn($event);
    }

    public function andThrowNotFound(): void
    {
        $this->expectation->andThrow(new EntityNotFoundException('Event not found'));
    }
}
