<?php

declare(strict_types=1);

namespace App\Application;

use App\Application\Bootloader\AppBootloader;
use App\Application\Bootloader\AttributesBootloader;
use App\Application\Bootloader\HttpHandlerBootloader;
use App\Application\Bootloader\MongoDBBootloader;
use App\Application\Bootloader\PersistenceBootloader;
use Modules\Inspector\Application\InspectorBootloader;
use Modules\Profiler\Application\ProfilerBootloader;
use Modules\Ray\Application\RayBootloader;
use Modules\HttpDumps\Application\HttpDumpsBootloader;
use Modules\Sentry\Application\SentryBootloader;
use Spiral\Boot\Bootloader\CoreBootloader;
use Spiral\Bootloader as Framework;
use Spiral\Cqrs\Bootloader\CqrsBootloader;
use Spiral\Cycle\Bootloader as CycleBridge;
use Spiral\Distribution\Bootloader\DistributionBootloader;
use Spiral\DotEnv\Bootloader\DotenvBootloader;
use Spiral\Events\Bootloader\EventsBootloader;
use Spiral\League\Event\Bootloader\EventBootloader;
use Spiral\Monolog\Bootloader\MonologBootloader;
use Spiral\Nyholm\Bootloader\NyholmBootloader;
use Spiral\RoadRunnerBridge\Bootloader as RoadRunnerBridge;
use Spiral\Stempler\Bootloader\StemplerBootloader;
use Spiral\Storage\Bootloader\StorageBootloader;
use Spiral\Tokenizer\Bootloader\TokenizerListenerBootloader;
use Spiral\Validation\Bootloader\ValidationBootloader;
use Spiral\Validator\Bootloader\ValidatorBootloader;

class Kernel extends \Spiral\Framework\Kernel
{
    protected const SYSTEM = [
        CoreBootloader::class,
        AttributesBootloader::class,
        TokenizerListenerBootloader::class,
        DotenvBootloader::class,
    ];

    protected function defineBootloaders(): array
    {
        return [
            Bootloader\ExceptionHandlerBootloader::class,

            // RoadRunner
            RoadRunnerBridge\CacheBootloader::class,
            RoadRunnerBridge\HttpBootloader::class,
            RoadRunnerBridge\CentrifugoBootloader::class,
            RoadRunnerBridge\QueueBootloader::class,
            RoadRunnerBridge\TcpBootloader::class,
            RoadRunnerBridge\LoggerBootloader::class,

            MonologBootloader::class,

            // Core Services
            Framework\SnapshotsBootloader::class,

            // Security and validation
            Framework\Security\EncrypterBootloader::class,
            Framework\Security\FiltersBootloader::class,
            ValidationBootloader::class,
            ValidatorBootloader::class,

            StemplerBootloader::class,

            // HTTP extensions
            NyholmBootloader::class,
            Framework\Http\RouterBootloader::class,
            Framework\Http\JsonPayloadsBootloader::class,

            // Databases
            CycleBridge\DatabaseBootloader::class,
            CycleBridge\MigrationsBootloader::class,

            // ORM
            CycleBridge\SchemaBootloader::class,
            CycleBridge\CycleOrmBootloader::class,
            CycleBridge\AnnotatedBootloader::class,

            // Event Dispatcher
            EventsBootloader::class,
            EventBootloader::class,
            CqrsBootloader::class,

            // Console commands
            Framework\CommandBootloader::class,
            CycleBridge\CommandBootloader::class,
            RoadRunnerBridge\CommandBootloader::class,

            // Configure route groups, middleware for route groups
            \Spiral\OpenApi\Bootloader\SwaggerBootloader::class,
            Bootloader\RoutesBootloader::class,

            StorageBootloader::class,
            DistributionBootloader::class,

            HttpHandlerBootloader::class,
            AppBootloader::class,
            InspectorBootloader::class,
            SentryBootloader::class,
            RayBootloader::class,
            HttpDumpsBootloader::class,
            ProfilerBootloader::class,
            MongoDBBootloader::class,
            PersistenceBootloader::class,
        ];
    }
}
