<?php

declare(strict_types=1);

namespace App\Interfaces\Console;

use App\Application\Database\CleanerInterface;
use Spiral\Boot\DirectoriesInterface;
use Spiral\Console\Attribute\AsCommand;
use Spiral\Console\Command;
use Spiral\Storage\Config\StorageConfig;
use Spiral\Storage\StorageInterface;

#[AsCommand(
    name: 'cleanup',
    description: 'Clean database and storage data',
)]
final class CleanupCommand extends Command
{
    public function __invoke(
        CleanerInterface $cleaner,
        StorageInterface $storage,
        DirectoriesInterface $dirs,
        StorageConfig $config,
    ): void {
        $this->info('Cleaning database...');

        foreach ($cleaner->clean() as $table) {
            $this->writeln(\sprintf('- Table %s cleaned', $table));
        }

        $this->newLine();

        $this->info('Cleaning storage...');
        foreach ($config->getAdapters() as $bucket => $adapter) {
            $adapter->deleteDirectory('*');
            $this->writeln(\sprintf('- Bucket %s cleaned', $bucket));
        }
    }
}
