<?php

declare(strict_types=1);

namespace App\Interfaces\Console;

use Cycle\Database\DatabaseInterface;
use Spiral\Console\Attribute\AsCommand;
use Spiral\Console\Command;

#[AsCommand(
    name: 'database:check-connection',
    description: 'Check database connection',
)]
final class CheckDatabaseConnectionCommand extends Command
{
    public function __invoke(DatabaseInterface $db): int
    {
        $tries = 0;
        $multiplier = 1.5;

        do {
            try {
                $db->getDriver()->connect();
                $this->info('Database connection is OK');
                break;
            } catch (\Throwable $e) {
                $tries++;
                $this->error($e->getMessage());
                $delay = (int) ($multiplier * $tries);
                $this->error('Cannot connect to the database. Retrying in ' . $delay . ' second...');
                \sleep($delay);
            }
        } while (true);

        return self::SUCCESS;
    }
}
