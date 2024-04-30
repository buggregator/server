<?php

declare(strict_types=1);

namespace Modules\Metrics\Interfaces\Console\Command;

use Modules\Metrics\Application\CollectorRepositoryInterface;
use Modules\Metrics\Application\MetricsDriverEnum;
use Spiral\Console\Attribute\AsCommand;
use Spiral\Console\Command;
use Spiral\RoadRunner\Metrics\MetricsInterface;

#[AsCommand(
    name: 'metrics:declare',
    description: 'Declare metrics',
)]
final class DeclareCommand extends Command
{
    public function __invoke(
        MetricsInterface $metrics,
        MetricsDriverEnum $driver,
        CollectorRepositoryInterface $repository,
    ): int {
        if ($driver === MetricsDriverEnum::Null) {
            $this->warning('Metrics driver is not available');
            return self::FAILURE;
        }

        foreach ($repository->findAll() as $name => $collector) {
            $this->info("Declaring metric: {$name}");

            try {
                $metrics->declare($name, $collector);
            } catch (\Throwable $e) {
                $this->error("Failed to declare metric: {$name}. Reason: {$e->getMessage()}");
            }
        }

        return self::SUCCESS;
    }
}
