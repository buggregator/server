<?php

declare(strict_types=1);

namespace Modules\Projects\Interfaces\Http\Resources;

use App\Application\HTTP\Response\JsonResource;
use Modules\Projects\Domain\Project;
use OpenApi\Attributes as OA;

/**
 * @property-read Project $data
 */
#[OA\Schema(
    schema: 'Event',
    properties: [
        new OA\Property(property: 'uuid', type: 'string', format: 'uuid'),
        new OA\Property(property: 'name', type: 'string'),
    ],
)]
final class ProjectResource extends JsonResource
{
    public function __construct(Project $data)
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
