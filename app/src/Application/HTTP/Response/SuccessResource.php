<?php

declare(strict_types=1);

namespace App\Application\HTTP\Response;

/**
 * @property-read bool $data
 */
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
