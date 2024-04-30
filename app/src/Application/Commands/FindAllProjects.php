<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Modules\Projects\Domain\Project;
use Spiral\Cqrs\QueryInterface;

/**
 * @implements QueryInterface<Project[]>
 */
final class FindAllProjects implements QueryInterface {}
