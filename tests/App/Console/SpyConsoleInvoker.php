<?php

declare(strict_types=1);

namespace Tests\App\Console;

use PHPUnit\Framework\TestCase;
use Spiral\Console\Command;
use Spiral\Core\InvokerInterface;

final class SpyConsoleInvoker implements InvokerInterface
{
    private array $runCommands = [];

    public function __construct(
        private readonly InvokerInterface $invoker,
        private readonly array $commandsToRun = [],
    ) {}

    public function invoke(callable|array|string $target, array $parameters = []): mixed
    {
        if (\is_array($target) && $target[0] instanceof Command) {
            $this->runCommands[] = $target[0]->getName();

            if (\in_array($target[0]->getName(), $this->commandsToRun, true)) {
                return $this->invoker->invoke($target, $parameters);
            }

            return Command::SUCCESS;
        }

        return $this->invoker->invoke($target, $parameters);
    }

    public function assertCommandRun(string $name): self
    {
        TestCase::assertContains($name, $this->runCommands, \sprintf('Command [%s] was not run.', $name));

        return $this;
    }

    public function assertCommandNotRun(string $name): self
    {
        TestCase::assertNotContains($name, $this->runCommands, \sprintf('Command [%s] was run.', $name));

        return $this;
    }
}
