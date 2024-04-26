<?php

declare(strict_types=1);

namespace Modules\Metrics\Infrastructure\RoadRunner;

use Modules\Metrics\Application\CollectorRegistryInterface;
use Modules\Metrics\Application\CollectorRepositoryInterface;
use Spiral\Core\Attribute\Singleton;
use Spiral\RoadRunner\Metrics\CollectorInterface;

#[Singleton]
final class Collector implements CollectorRepositoryInterface, CollectorRegistryInterface
{
    /** @var array<non-empty-string, CollectorInterface> */
    private array $collectors = [];

    public function register(string $name, CollectorInterface $collector): void
    {
        $this->collectors[$name] = $collector;
    }

    public function findAll(): iterable
    {
        return $this->collectors;
    }
}
