<?php

declare(strict_types=1);

namespace Modules\Smtp\Interfaces\Http\Resources;

use App\Application\HTTP\Response\JsonResource;
use JsonSerializable;
use Modules\Smtp\Domain\Attachment;
use OpenApi\Attributes as OA;

/**
 * @property-read Attachment $data
 */
#[OA\Schema(
    schema: 'SmtpAttachment',
    properties: [
        new OA\Property(property: 'uuid', type: 'string', format: 'uuid'),
        new OA\Property(property: 'name', type: 'string'),
        new OA\Property(property: 'path', type: 'string'),
        new OA\Property(property: 'size', type: 'integer'),
        new OA\Property(property: 'mime', type: 'string'),
    ],
)]
final class AttachmentResource extends JsonResource
{
    public function __construct(Attachment $data)
    {
        parent::__construct($data);
    }

    protected function mapData(): array|JsonSerializable
    {
        return [
            'uuid' => (string) $this->data->getUuid(),
            'name' => $this->data->getFilename(),
            'path' => $this->data->getPath(),
            'size' => $this->data->getSize(),
            'mime' => $this->data->getMime(),
        ];
    }
}
