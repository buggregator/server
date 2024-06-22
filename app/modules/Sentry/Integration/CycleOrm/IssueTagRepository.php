<?php

declare(strict_types=1);

namespace Modules\Sentry\Integration\CycleOrm;

use Cycle\ORM\Select\Repository;
use Modules\Sentry\Domain\IssueTagRepositoryInterface;

final class IssueTagRepository extends Repository implements IssueTagRepositoryInterface
{

}
