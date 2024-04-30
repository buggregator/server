<?php

declare(strict_types=1);

namespace App\Interfaces\Console;

use App\Application\Commands\HandleReceivedEvent;
use Spiral\Console\Attribute\AsCommand;
use Spiral\Console\Command;
use Spiral\Cqrs\CommandBusInterface;

#[AsCommand(
    name: 'test',
    description: 'Test command',
)]
final class TestCommand extends Command
{
    public function __invoke(CommandBusInterface $bus)
    {
        $bus->dispatch(
            new HandleReceivedEvent(
                'sentry',
                [
                    'platform' => 'php',
                    'environment' => 'testing',
                    'server_name' => 'localhost',
                    'message' => 'Test message',
                    'event_id' => '1234567890',
                ],
            ),
        );
    }
}
