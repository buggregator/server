<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Persistence\DriverEnum;
use App\Interfaces\Console\RegisterModulesCommand;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Console\Bootloader\ConsoleBootloader;
use Spiral\Cycle\Bootloader as CycleBridge;

final class PersistenceBootloader extends Bootloader
{
    public function defineDependencies(): array
    {
        return [
            // Databases
            CycleBridge\DatabaseBootloader::class,
            CycleBridge\MigrationsBootloader::class,

            // ORM
            CycleBridge\SchemaBootloader::class,
            CycleBridge\CycleOrmBootloader::class,
            CycleBridge\AnnotatedBootloader::class,
        ];
    }

    public function init(ConsoleBootloader $console, DriverEnum $driver): void
    {
        if ($driver === DriverEnum::Database) {
            $console->addSequence(
                name: RegisterModulesCommand::SEQUENCE,
                sequence: 'migrate',
                header: 'Migration',
            );
        }
    }
}
