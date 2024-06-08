<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Http\Controllers;

use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Profiler\Application\Query\FindCallGraphByUuid;
use Modules\Profiler\Interfaces\Http\Request\CallGraphRequest;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Router\Annotation\Route;

final readonly class ShowCallGraphAction
{
    #[Route(route: 'profiler/<uuid>/call-graph', name: 'profiler.show.call-graph', methods: ['GET'], group: 'api')]
    public function __invoke(
        CallGraphRequest $request,
        QueryBusInterface $bus,
        Uuid $uuid,
    ): array {
        try {
            /** @var array $callGraph */
            $callGraph = $bus->ask(
                new FindCallGraphByUuid(
                    profileUuid: $uuid,
                    threshold: $request->threshold,
                    percentage: $request->percentage,
                    metric: $request->metric,
                ),
            );
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }

        return $callGraph;
    }
}
