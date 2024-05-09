<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Http\Resources;

use App\Application\HTTP\Response\JsonResource;
use Modules\Webhooks\Domain\Delivery;
use OpenApi\Attributes as OA;

/**
 * @property-read Delivery $data
 */
#[OA\Schema(
    schema: 'Delivery',
    properties: [
        new OA\Property(property: 'uuid', type: 'string', format: 'uuid'),
        new OA\Property(property: 'payload', type: 'string'),
        new OA\Property(property: 'response', type: 'string'),
        new OA\Property(property: 'status', type: 'integer'),
        new OA\Property(property: 'created_at', type: 'string', format: 'date-time'),
    ],
)]
final class DeliveryResource extends JsonResource
{
    public function __construct(Delivery $data)
    {
        parent::__construct($data);
    }

    protected function mapData(): array|\JsonSerializable
    {
        return [
            'uuid' => (string) $this->data->getUuid(),
            'payload' => $this->data->getPayload(),
            'response' => $this->data->getResponse(),
            'status' => $this->data->getStatus(),
            'created_at' => $this->data->getCreatedAt()->format(\DateTimeInterface::W3C),
        ];
    }
}
