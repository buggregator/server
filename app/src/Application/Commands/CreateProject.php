<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Modules\Projects\Domain\Project;
use Spiral\Cqrs\CommandInterface;

/**
 * @implements CommandInterface<Project>
 */
final readonly class CreateProject implements CommandInterface
{
    public function __construct(
        public string $key,
        public string $name,
    ) {}
}
