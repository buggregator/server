<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Http\Controllers;

use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Profiler\Application\Query\FindProfileSummaryByUuid;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Router\Annotation\Route;

final readonly class ShowSummaryAction
{
    #[Route(route: 'profiler/<uuid>/summary', name: 'profiler.show.summary', methods: ['GET'], group: 'api')]
    public function __invoke(
        QueryBusInterface $bus,
        Uuid $uuid,
    ): array {
        try {
            return $bus->ask(new FindProfileSummaryByUuid(profileUuid: $uuid));
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }
    }
}
