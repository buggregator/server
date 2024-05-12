<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Http\Controllers;

use App\Application\Commands\FindWebhooks;
use Modules\Webhooks\Interfaces\Http\Resources\WebhookCollection;
use Modules\Webhooks\Interfaces\Http\Resources\WebhookResource;
use OpenApi\Attributes as OA;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Router\Annotation\Route;

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
final readonly class ListAction
{
    #[Route(route: 'webhooks', name: 'webhooks.list', methods: 'GET', group: 'api')]
    public function __invoke(
        QueryBusInterface $bus,
    ): WebhookCollection {
        $webhooks = $bus->ask(new FindWebhooks());

        return new WebhookCollection($webhooks);
    }
}
