<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use MongoDB\Client;
use MongoDB\Database;
use MongoDB\Driver\ServerApi;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\EnvironmentInterface;

final class MongoDBBootloader extends Bootloader
{
    protected const SINGLETONS = [
        Client::class => [self::class, 'createClient'],
        Database::class => [self::class, 'selectDatabase'],
    ];

    private function createClient(EnvironmentInterface $env): Client
    {
        return new Client(
            $env->get('MONGODB_CONNECTION')
        );
    }

    private function selectDatabase(Client $client, EnvironmentInterface $env): Database
    {
        $database = $client->selectDatabase($env->get('MONGODB_DATABASE'));
        $database->command(['ping' => 1]);

        return $database;
    }
}
