<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Http\Controllers;

use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Profiler\Application\Query\FindDigraphByUuid;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Router\Annotation\Route;

final readonly class ShowDigraphAction
{
    #[Route(route: 'profiler/<uuid>/digraph', name: 'profiler.show.digraph', methods: ['GET'], group: 'api')]
    public function __invoke(
        QueryBusInterface $bus,
        Uuid $uuid,
    ): string {
        try {
            /** @var string $digraph */
            $digraph = $bus->ask(
                new FindDigraphByUuid(
                    profileUuid: $uuid,
                ),
            );
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }

        return $digraph;
    }
}
