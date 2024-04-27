<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Resources;

use App\Application\HTTP\Response\JsonResource;
use Modules\Events\Domain\Event;
use OpenApi\Attributes as OA;

/**
 * @property-read Event $data
 */
#[OA\Schema(
    schema: 'Event',
    properties: [
        new OA\Property(property: 'uuid', type: 'string', format: 'uuid'),
        new OA\Property(property: 'project', description: 'Project', type: 'string', format: 'uuid'),
        new OA\Property(property: 'type', type: 'string'),
        new OA\Property(property: 'payload', description: 'Event payload based on type', type: 'object'),
        new OA\Property(property: 'timestamp', type: 'float', example: 1630540800.12312),
    ],
)]
final class EventResource extends JsonResource
{
    public function __construct(Event $data)
    {
        parent::__construct($data);
    }

    protected function mapData(): array|\JsonSerializable
    {
        return [
            'uuid' => (string)$this->data->getUuid(),
            'project' => $this->data->getProject(),
            'type' => $this->data->getType(),
            'payload' => $this->data->getPayload(),
            'timestamp' => $this->data->getTimestamp(),
        ];
    }
}
