<?php

declare(strict_types=1);

namespace Modules\Metrics\Application;

use App\Application\Mode;
use Cycle\ORM\FactoryInterface;
use Modules\Metrics\Infrastructure\RoadRunner\Collector;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Console\Bootloader\ConsoleBootloader;
use Spiral\Goridge\RPC\RPCInterface;
use Spiral\RoadRunner\Metrics\MetricsFactory;
use Spiral\RoadRunner\Metrics\MetricsInterface;
use Spiral\RoadRunner\Metrics\MetricsOptions;
use Spiral\RoadRunnerBridge\Bootloader\RoadRunnerBootloader;

final class MetricsBootloader extends Bootloader
{
    public function defineDependencies(): array
    {
        return [
            RoadRunnerBootloader::class,
            MetricsBootloader::class,
        ];
    }

    public function defineSingletons(): array
    {
        return [
            MetricsInterface::class => static function (
                MetricsDriverEnum $driver,
                FactoryInterface $factory,
                RPCInterface $rpc,
            ) {
                return match ($driver) {
                    MetricsDriverEnum::RoadRunner => (new MetricsFactory())->create(
                        $rpc,
                        new MetricsOptions(
                            retryAttempts: 2,
                            retrySleepMicroseconds: 300,
                            suppressExceptions: true,
                        ),
                    ),
                    MetricsDriverEnum::Null => new NullDriver(),
                };
            },

            CollectorRepositoryInterface::class => Collector::class,
            CollectorRegistryInterface::class => Collector::class,
            Collector::class => Collector::class,
        ];
    }

    public function init(ConsoleBootloader $console): void
    {
        $console->addConfigureSequence(
            sequence: 'metrics:declare',
            header: 'Declare metrics',
        );
    }
}
