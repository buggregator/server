<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Modules\Projects\Domain\ProjectInterface;
use Spiral\Cqrs\QueryInterface;

/**
 * @implements QueryInterface<ProjectInterface[]>
 */
final class FindAllProjects implements QueryInterface {}
