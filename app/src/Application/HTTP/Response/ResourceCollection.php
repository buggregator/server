<?php

declare(strict_types=1);

namespace App\Application\HTTP\Response;

use Psr\Http\Message\ResponseInterface;
use Spiral\DataGrid\GridInterface;
use Spiral\Http\Traits\JsonTrait;

class ResourceCollection implements ResourceInterface
{
    use JsonTrait;

    private readonly array $args;

    /**
     * @param class-string<ResourceInterface> $resourceClass
     */
    public function __construct(
        protected readonly iterable $data,
        protected string|\Closure $resourceClass = JsonResource::class,
        mixed ...$args
    ) {
        $this->args = $args;
    }

    /**
     * @return class-string<ResourceInterface>|\Closure
     */
    protected function getResourceClass(): string|\Closure
    {
        return $this->resourceClass;
    }

    protected function getData(): iterable
    {
        return $this->data;
    }

    public function jsonSerialize(): array
    {
        $data = [];
        $resource = $this->getResourceClass();

        foreach ($this->getData() as $key => $row) {
            if ($row instanceof \JsonSerializable) {
                $data[$key] = $row;
                continue;
            }

            if (\is_string($resource)) {
                $resource = static fn(mixed $row, mixed ...$args): ResourceInterface => new $resource($row, ...$args);
            }

            $data[$key] = $resource($row, ...$this->args);
        }

        return $this->wrapData($data);
    }

    public function toResponse(ResponseInterface $response): ResponseInterface
    {
        return $this->writeJson($response, $this);
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
