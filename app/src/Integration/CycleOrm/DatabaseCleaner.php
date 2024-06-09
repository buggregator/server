<?php

declare(strict_types=1);

namespace App\Integration\CycleOrm;

use App\Application\Database\CleanerInterface;
use Cycle\Database\DatabaseProviderInterface;
use Cycle\Migrations\Config\MigrationConfig;

final readonly class DatabaseCleaner implements CleanerInterface
{
    public function __construct(
        private MigrationConfig $config,
        private DatabaseProviderInterface $provider,
    ) {}

    public function clean(?string $database = null): \Generator
    {
        $db = $this->provider->database($database);


        foreach ($db->getTables() as $table) {
            $this->disableForeignKeyConstraints($database);

            // Skip the table that stores the list of executed migrations
            if ($table->getName() === $this->config->getTable()) {
                continue;
            }

            /**
             * @psalm-suppress UndefinedInterfaceMethod
             */
            $db->getDriver()->getSchemaHandler()->eraseTable($db->table($table->getFullName())->getSchema());
            yield $table->getName();

            $this->enableForeignKeyConstraints($database);
        }
    }

    public function disableForeignKeyConstraints(?string $database = null): void
    {
        $db = $this->provider->database($database);

        /**
         * @psalm-suppress UndefinedInterfaceMethod
         */
        $db->getDriver()->getSchemaHandler()->disableForeignKeyConstraints();
    }

    public function enableForeignKeyConstraints(?string $database = null): void
    {
        $db = $this->provider->database($database);

        /**
         * @psalm-suppress UndefinedInterfaceMethod
         */
        $db->getDriver()->getSchemaHandler()->enableForeignKeyConstraints();
    }
}
