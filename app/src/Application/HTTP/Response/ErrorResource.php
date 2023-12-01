<?php

declare(strict_types=1);

namespace App\Application\HTTP\Response;

use App\Application\Exception\EntityNotFoundException;
use Spiral\Http\Exception\ClientException;

/**
 * @property-read \Throwable $data
 */
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
