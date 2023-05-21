<?php

declare(strict_types=1);

namespace App\Interfaces\Console;

use Buggregator\Client\Application;
use Buggregator\Client\Sender;
use Spiral\Console\Attribute\AsCommand;
use Spiral\Console\Attribute\Option;
use Spiral\Console\Command;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;
use Symfony\Component\Console\Style\SymfonyStyle;

#[AsCommand(name: 'dump:server')]
final class DumpServerCommand extends Command
{
    #[Option(shortcut: 'p', description: 'Port to listen')]
    public int $port = 9912;

    #[Option(shortcut: 's', description: 'Data senders')]
    public array $senders = ['internal', 'console'];

    public function __invoke(Sender\SenderRegistry $registry, Application $app): int
    {
        $registry->register('console', Sender\ConsoleSender::create($this->output));

        $app->run(
            senders: $registry->getSenders($this->senders)
        );

        return self::SUCCESS;
    }

    protected function prepareOutput(InputInterface $input, OutputInterface $output): OutputInterface
    {
        $output->setDecorated(true);

        return new SymfonyStyle($input, $output);
    }
}
