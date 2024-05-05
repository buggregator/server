<?php

declare(strict_types=1);

namespace App\Application\HTTP\Response;

use App\Application\Exception\EntityNotFoundException;
use Spiral\Http\Exception\ClientException;
use OpenApi\Attributes as OA;

/**
 * @property-read \Throwable $data
 * todo: add swagger attributes
 */
#[OA\Schema(
    schema: 'ErrorResource',
    properties: [
        new OA\Property(property: 'message', type: 'string'),
        new OA\Property(property: 'code', type: 'integer'),
    ],
    type: 'object',
)]
final class ErrorResource extends JsonResource
{
    public function __construct(\Throwable $data)
    {
        parent::__construct($data);
    }

    protected function mapData(): array|\JsonSerializable
    {
        return [
            'message' => $this->data->getMessage(),
            'code' => $this->getCode(),
        ];
    }

    protected function getCode(): int
    {
        return match (true) {
            $this->data instanceof EntityNotFoundException => 404,
            $this->data instanceof ClientException => $this->data->getCode(),
            default => 500,
        };
    }
}
