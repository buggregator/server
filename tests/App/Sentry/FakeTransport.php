<?php

declare(strict_types=1);

namespace Tests\App\Sentry;

use Sentry\Event;
use Sentry\EventId;
use Sentry\Serializer\PayloadSerializerInterface;
use Sentry\Transport\Result;
use Sentry\Transport\ResultStatus;
use Sentry\Transport\TransportInterface;

final class FakeTransport implements TransportInterface
{
    /** @var array<non-empty-string, non-empty-string> */
    private array $events = [];

    public function __construct(
        private readonly PayloadSerializerInterface $payloadSerializer,
    ) {}

    public function send(Event $event): Result
    {
        $this->events[(string) $event->getId()] = $this->payloadSerializer->serialize($event);

        return new Result(ResultStatus::success(), $event);
    }

    public function close(?int $timeout = null): Result
    {
        return new Result(ResultStatus::success());
    }

    public function findEvent(EventId $id): string
    {
        return $this->events[(string) $id] ?? throw new \InvalidArgumentException('Event not found');
    }
}
