<?php

declare(strict_types=1);

namespace Modules\Projects\Interfaces\Http\Controllers;

use App\Application\Commands\FindAllProjects;
use Modules\Projects\Interfaces\Http\Resources\ProjectCollection;
use Modules\Projects\Interfaces\Http\Resources\ProjectResource;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Router\Annotation\Route;
use OpenApi\Attributes as OA;

#[OA\Get(
    path: '/api/projects',
    description: 'Retrieve all projects',
    tags: ['Projects'],
    responses: [
        new OA\Response(
            response: 200,
            description: 'Success',
            content: new OA\JsonContent(
                properties: [
                    new OA\Property(
                        property: 'data',
                        type: 'array',
                        items: new OA\Items(
                            ref: ProjectResource::class,
                        ),
                    ),
                    new OA\Property(
                        property: 'meta',
                        ref: '#/components/schemas/ResponseMeta',
                        type: 'object',
                    ),
                ],
            ),
        ),
        new OA\Response(
            response: 404,
            description: 'Not found',
            content: new OA\JsonContent(
                ref: '#/components/schemas/NotFoundError',
            ),
        ),
    ],
)]
final class ListAction
{
    #[Route(route: 'projects', name: 'projects.list', methods: 'GET', group: 'api')]
    public function __invoke(QueryBusInterface $bus): ProjectCollection
    {
        return new ProjectCollection(
            $bus->ask(
                new FindAllProjects(),
            ),
        );
    }
}
