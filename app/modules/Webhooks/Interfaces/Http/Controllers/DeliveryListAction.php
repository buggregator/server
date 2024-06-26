<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Http\Controllers;

use App\Application\Commands\FindWebhookDeliveries;
use App\Application\Domain\ValueObjects\Uuid;
use Modules\Webhooks\Interfaces\Http\Resources\DeliveryCollection;
use Modules\Webhooks\Interfaces\Http\Resources\DeliveryResource;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Router\Annotation\Route;
use OpenApi\Attributes as OA;

#[OA\Get(
    path: '/api/webhook/{uuid}/deliveries',
    description: 'Retrieve all webhook deliveries',
    tags: ['Webhooks'],
    parameters: [
        new OA\PathParameter(
            name: 'uuid',
            description: 'Webhook UUID',
            required: true,
            schema: new OA\Schema(type: 'string', format: 'uuid'),
        ),
    ],
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
                            ref: DeliveryResource::class,
                        ),
                    ),
                ],
            ),
        ),
    ],
)]
final readonly class DeliveryListAction
{
    #[Route(route: 'webhook/<uuid>/deliveries', name: 'webhooks.delivery.list', methods: 'GET', group: 'api')]
    public function __invoke(
        QueryBusInterface $bus,
        Uuid $uuid,
    ): DeliveryCollection {
        $deliveries = $bus->ask(new FindWebhookDeliveries(webhookUuid: $uuid));

        return new DeliveryCollection($deliveries);
    }
}
