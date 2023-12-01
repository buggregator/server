<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use MongoDB\Client;
use MongoDB\Database;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\EnvironmentInterface;

final class MongoDBBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            Client::class => static function (EnvironmentInterface $env): Client {
                return new Client(
                    $env->get('MONGODB_CONNECTION'),
                );
            },
            Database::class => static function (Client $client, EnvironmentInterface $env): Database {
                $database = $client->selectDatabase($env->get('MONGODB_DATABASE'));
                $database->command(['ping' => 1]);

                return $database;
            },
        ];
    }
}
