<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Resources;

use App\Application\HTTP\Response\JsonResource;
use OpenApi\Attributes as OA;

/**
 * @property-read array{type: string, cnt: int|string} $data
 */
#[OA\Schema(
    schema: 'EventTypeCount',
    properties: [
        new OA\Property(property: 'type', type: 'string'),
        new OA\Property(property: 'count', type: 'integer'),
    ],
)]
final class EventTypeCountResource extends JsonResource
{
    protected function mapData(): array|\JsonSerializable
    {
        return [
            'type' => $this->data['type'],
            'count' => (int) $this->data['cnt'],
        ];
    }
}
