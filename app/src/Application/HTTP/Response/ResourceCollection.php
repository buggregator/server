<?php

declare(strict_types=1);

namespace App\Application\HTTP\Response;

use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\DataGrid\GridInterface;
use Spiral\Http\Traits\JsonTrait;

class ResourceCollection implements ResourceInterface
{
    use JsonTrait;

    /**
     * @param class-string<ResourceInterface> $resourceClass
     */
    public function __construct(
        protected readonly iterable $data,
        protected string $resourceClass = JsonResource::class
    ) {
    }

    /**
     * @return class-string<ResourceInterface>
     */
    protected function getResourceClass(): string
    {
        return $this->resourceClass;
    }

    protected function getData(): iterable
    {
        return $this->data;
    }

    public function resolve(ServerRequestInterface $request): array
    {
        $data = [];
        $resourceClass = $this->getResourceClass();

        foreach ($this->getData() as $key => $row) {
            $data[$key] = (new $resourceClass($row))->resolve($request);
        }

        return $this->wrapData($data);
    }

    public function toResponse(ServerRequestInterface $request, ResponseInterface $response): ResponseInterface
    {
        return $this->writeJson($response, $this->resolve($request));
    }

    protected function wrapData(array $data): array
    {
        $grid = [];

        if ($this->data instanceof GridInterface) {
            foreach ([GridInterface::FILTERS, GridInterface::SORTERS] as $key) {
                $grid[$key] = $this->data->getOption($key);
            }
        }

        return [
            'data' => $data,
            'meta' => [
                'grid' => $grid,
            ],
        ];
    }
}
