<?php

declare(strict_types=1);

namespace App\Application\HTTP\Response;

use OpenApi\Attributes as OA;

/**
 * @property-read bool $data
 */
#[OA\Schema(
    schema: 'SuccessResource',
    properties: [
        new OA\Property(property: 'status', type: 'boolean'),
    ],
    type: 'object'
)]
final class SuccessResource extends JsonResource
{
    public function __construct(bool $status = true)
    {
        parent::__construct($status);
    }

    protected function mapData(): array|\JsonSerializable
    {
        return [
            'status' => $this->data,
        ];
    }
}
