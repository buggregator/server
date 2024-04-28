<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Modules\Projects\Domain\Project;
use Spiral\Cqrs\QueryInterface;

/**
 * @implements QueryInterface<Project|null>
 */
final readonly class FindProjectByKey implements QueryInterface
{
    public function __construct(
        public string $key
    ) {}
}
