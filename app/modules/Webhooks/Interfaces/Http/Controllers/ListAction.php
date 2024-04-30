<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Http\Controllers;

use Modules\Webhooks\Domain\WebhookRepositoryInterface;
use Modules\Webhooks\Interfaces\Http\Resources\WebhookCollection;
use Modules\Webhooks\Interfaces\Http\Resources\WebhookResource;
use Spiral\Router\Annotation\Route;
use OpenApi\Attributes as OA;

#[OA\Get(
    path: '/api/webhooks',
    description: 'Retrieve all webhooks',
    tags: ['Webhooks'],
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
                            ref: WebhookResource::class,
                        ),
                    ),
                ],
            ),
        ),
    ],
)]
final class ListAction
{
    #[Route(route: 'webhooks', name: 'webhooks.list', methods: 'GET', group: 'api')]
    public function __invoke(
        WebhookRepositoryInterface $repository,
    ): WebhookCollection {
        return new WebhookCollection(
            $repository->findAll(),
        );
    }
}
