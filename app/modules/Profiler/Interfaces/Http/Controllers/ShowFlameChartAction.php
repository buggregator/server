<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Http\Controllers;

use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Profiler\Application\Query\FindFlameChartByUuid;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Router\Annotation\Route;

final readonly class ShowFlameChartAction
{
    #[Route(route: 'profiler/<uuid>/flame-chart', name: 'profiler.show.flame-chart', methods: ['GET'], group: 'api')]
    public function __invoke(QueryBusInterface $bus, Uuid $uuid): array
    {
        try {
            /** @var array $flameChart */
            $flameChart = $bus->ask(new FindFlameChartByUuid($uuid));
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }

        return $flameChart;
    }
}
