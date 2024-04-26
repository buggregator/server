<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Http\Resources;

use App\Application\HTTP\Response\JsonResource;
use Modules\Webhooks\Domain\Webhook;
use OpenApi\Attributes as OA;

/**
 * @property-read Webhook $data
 */
#[OA\Schema(
    schema: 'Webhook',
    properties: [
        new OA\Property(property: 'uuid', type: 'string', format: 'uuid'),
        new OA\Property(property: 'event', type: 'string'),
        new OA\Property(property: 'url', type: 'string'),
        new OA\Property(property: 'headers', type: 'array', items: new OA\Items(type: 'string')),
        new OA\Property(property: 'verify_ssl', type: 'boolean'),
        new OA\Property(property: 'retry_on_failure', type: 'boolean'),
    ],
)]
final class WebhookResource extends JsonResource
{
    public function __construct(Webhook $data)
    {
        parent::__construct($data);
    }

    protected function mapData(): array|\JsonSerializable
    {
        return [
            'uuid' => (string)$this->data->uuid,
            'event' => $this->data->event,
            'url' => $this->data->url,
            'headers' => $this->data->headers,
            'verify_ssl' => $this->data->verifySsl,
            'retry_on_failure' => $this->data->retryOnFailure,
        ];
    }
}
