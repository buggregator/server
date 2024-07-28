<?php

declare(strict_types=1);

namespace Modules\Projects\Interfaces\Http\Resources;

use App\Application\HTTP\Response\JsonResource;
use Modules\Projects\Domain\ProjectInterface;
use OpenApi\Attributes as OA;

/**
 * @property-read ProjectInterface $data
 */
#[OA\Schema(
    schema: 'Project',
    properties: [
        new OA\Property(property: 'key', type: 'string'),
        new OA\Property(property: 'name', type: 'string'),
    ],
)]
final class ProjectResource extends JsonResource
{
    public function __construct(ProjectInterface $data)
    {
        parent::__construct($data);
    }

    protected function mapData(): array|\JsonSerializable
    {
        return [
            'key' => $this->data->getKey(),
            'name' => $this->data->getName(),
        ];
    }
}
