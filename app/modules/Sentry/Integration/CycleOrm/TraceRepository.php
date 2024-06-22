<?php

declare(strict_types=1);

namespace Modules\Sentry\Integration\CycleOrm;

use Cycle\ORM\Select\Repository;
use Modules\Sentry\Domain\TraceRepositoryInterface;

final class TraceRepository extends Repository implements TraceRepositoryInterface
{

}
