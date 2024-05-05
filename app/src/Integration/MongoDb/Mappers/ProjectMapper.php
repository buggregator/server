<?php

declare(strict_types=1);

namespace App\Integration\MongoDb\Mappers;

use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ValueObject\Key;
use MongoDB\Model\BSONDocument;

final readonly class ProjectMapper
{
    public function toProject(BSONDocument $document): Project
    {
        /** @psalm-suppress InternalMethod */
        return new Project(
            new Key($document['_id']),
            $document['name'],
        );
    }

    public function toDocument(Project $project): array
    {
        return [
            '_id' => (string) $project->getKey(),
            'name' => $project->getName(),
        ];
    }
}
