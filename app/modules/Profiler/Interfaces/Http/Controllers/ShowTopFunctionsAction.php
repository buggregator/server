<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Http\Controllers;

use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Profiler\Application\Query\FindCallGraphByUuid;
use Modules\Profiler\Application\Query\FindTopFunctionsByUuid;
use Modules\Profiler\Interfaces\Http\Request\CallGraphRequest;
use Modules\Profiler\Interfaces\Http\Request\TopFunctionsRequest;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Router\Annotation\Route;

final readonly class ShowTopFunctionsAction
{
    #[Route(route: 'profiler/<uuid>/top', name: 'profiler.show.top', methods: ['GET'], group: 'api')]
    public function __invoke(
        TopFunctionsRequest $request,
        QueryBusInterface $bus,
        Uuid $uuid,
    ): array {
        try {
            /** @var array $functions */
            $functions = $bus->ask(
                new FindTopFunctionsByUuid(
                    profileUuid: $uuid,
                    limit: $request->limit,
                    metric: $request->metric,
                ),
            );
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }

        return $functions;
    }
}
