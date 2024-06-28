<?php

declare(strict_types=1);

namespace App\Application\HTTP\Response;

use JsonSerializable;
use Psr\Http\Message\ResponseInterface;
use Spiral\Http\Traits\JsonTrait;

class JsonResource implements ResourceInterface
{
    use JsonTrait;

    public function __construct(protected readonly mixed $data = [])
    {
    }

    protected function mapData(): array|JsonSerializable
    {
        return $this->data;
    }

    protected function getCode(): int
    {
        return 200;
    }

    public function toResponse(ResponseInterface $response): ResponseInterface
    {
        return $this->writeJson($response, $this, $this->getCode());
    }

    protected function wrapData(array $data): array
    {
        return $data;
    }

    public function jsonSerialize(): array
    {
        $data = $this->mapData();

        if ($data instanceof JsonSerializable) {
            $data = $data->jsonSerialize();
        }

        foreach ($data as $key => $value) {
            if ($value instanceof ResourceInterface) {
                $data[$key] = $value->jsonSerialize();
            }
        }

        return $this->wrapData($data);
    }
}
