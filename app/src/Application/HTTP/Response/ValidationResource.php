<?php

declare(strict_types=1);

namespace App\Application\HTTP\Response;

use Spiral\Filters\Exception\ValidationException;

/**
 * @property-read ValidationException $data
 */
final class ValidationResource extends JsonResource
{
    public function __construct(ValidationException $data)
    {
        parent::__construct($data);
    }

    protected function mapData(): array|\JsonSerializable
    {
        return [
            'message' => $this->data->getMessage(),
            'code' => $this->getCode(),
            'errors' => $this->data->errors,
            'context' => $this->data->context,
        ];
    }

    protected function getCode(): int
    {
        return $this->data->getCode();
    }
}
