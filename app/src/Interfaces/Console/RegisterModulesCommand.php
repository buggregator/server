<?php

declare(strict_types=1);

namespace App\Interfaces\Console;

use App\Application\Commands\HandleReceivedEvent;
use Psr\Container\ContainerInterface;
use Spiral\Console\Attribute\AsCommand;
use Spiral\Console\Command\SequenceCommand;
use Spiral\Console\Config\ConsoleConfig;
use Spiral\Cqrs\CommandBusInterface;

#[AsCommand(
    name: 'register:modules',
    description: 'Register modules',
)]
final class RegisterModulesCommand extends SequenceCommand
{
    public const SEQUENCE = 'register:modules';

    public function __invoke(ConsoleConfig $config, ContainerInterface $container): int
    {
        $this->info('Register buggregator modules...');
        $this->newLine();

        return $this->runSequence($config->getSequence(self::SEQUENCE), $container);
    }
}
