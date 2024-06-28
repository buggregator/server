<?php

declare(strict_types=1);

namespace Tests\App\Broadcasting;

use App\Application\Broadcasting\InMemoryDriver;
use PHPUnit\Framework\TestCase;
use Spiral\Core\Container;

final readonly class BroadcastFaker
{
    public function __construct(
        private Container $container,
    ) {}

    public function dump(): self
    {
        dump($this->getMessages());

        return $this;
    }

    public function reset(): self
    {
        $this->container->get(InMemoryDriver::class)->reset();

        return $this;
    }

    public function assertPushedTimes(string|\Stringable $topic, int $times = 1): array
    {
        $messages = $this->filterMessages((string) $topic);

        TestCase::assertCount(
            $times,
            $messages,
            \sprintf(
                'The expected message in topic [%s] was sent {%d} times instead of {%d} times.',
                $topic,
                \count($messages),
                $times,
            ),
        );

        return $messages;
    }


    public function assertPushed(string|\Stringable $topic, \Closure $callback = null): self
    {
        $messages = $this->filterMessages((string) $topic, $callback);

        TestCase::assertTrue(
            $messages !== [],
            \sprintf('The expected message [%s] was not pushed.', $topic),
        );

        return $this;
    }

    public function assertNotPushed(string|\Stringable $topic, \Closure $callback = null): self
    {
        $messages = $this->filterMessages((string) $topic, $callback);

        TestCase::assertCount(
            0,
            $messages,
            \sprintf('The unexpected message [%s] was pushed.', $topic),
        );

        return $this;
    }

    public function assertNothingPushed(): self
    {
        $pushedMessages = $this->getMessages();
        $messages = \implode(', ', \array_keys($this->getMessages()));

        TestCase::assertCount(
            0,
            $pushedMessages,
            \sprintf('The following messages were pushed unexpectedly in the following topics: %s', $messages),
        );

        return $this;
    }

    private function getMessages(): array
    {
        return $this->container->get(InMemoryDriver::class)->published();
    }

    private function filterMessages(string $topic, \Closure $callback = null): array
    {
        $messages = $this->getMessages()[$topic] ?? [];

        $callback = $callback ?: static fn(array $data): bool => true;

        return \array_filter($messages, static fn(array $data) => $callback($data));
    }
}
