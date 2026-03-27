<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Http\Controllers;

use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Profiler\Application\Query\CompareProfiles;
use Modules\Profiler\Interfaces\Http\Request\CompareProfilesRequest;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Router\Annotation\Route;

final readonly class CompareProfilesAction
{
    #[Route(route: 'profiler/compare', name: 'profiler.compare', methods: ['GET'], group: 'api')]
    public function __invoke(
        CompareProfilesRequest $request,
        QueryBusInterface $bus,
    ): array {
        try {
            return $bus->ask(
                new CompareProfiles(
                    baseProfileUuid: Uuid::fromString($request->base),
                    compareProfileUuid: Uuid::fromString($request->compare),
                ),
            );
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }
    }
}
