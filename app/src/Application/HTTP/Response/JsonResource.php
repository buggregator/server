<?php

declare(strict_types=1);

namespace App\Application\HTTP\Response;

use JsonSerializable;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Http\Traits\JsonTrait;

class JsonResource implements ResourceInterface, \ArrayAccess
{
    use JsonTrait;

    public function __construct(
        protected readonly array|JsonSerializable $data
    ) {
    }

    protected function mapData(ServerRequestInterface $request): array|JsonSerializable
    {
        return $this->data;
    }

    public function resolve(ServerRequestInterface $request): array
    {
        $data = $this->mapData($request);

        if ($data instanceof JsonSerializable) {
            $data = $data->jsonSerialize();
        }

        return $this->wrapData($data);
    }

    public function toResponse(ServerRequestInterface $request, ResponseInterface $response): ResponseInterface
    {
        return $this->writeJson($response, $this->resolve($request));
    }

    protected function wrapData(array $data): array
    {
        return $data;
    }

    public function offsetExists(mixed $offset): bool
    {
        return isset($this->data[$offset]);
    }

    public function offsetGet(mixed $offset): mixed
    {
        return $this->data[$offset];
    }

    public function offsetSet(mixed $offset, mixed $value): void
    {
        throw new \RuntimeException('Resource is read-only');
    }

    public function offsetUnset(mixed $offset): void
    {
        throw new \RuntimeException('Resource is read-only');
    }

    public function __isset($key)
    {
        return $this->offsetExists($key);
    }

    public function __get($key)
    {
        return $this->offsetGet($key);
    }
}
