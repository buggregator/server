<?php

declare(strict_types=1);

namespace Modules\Sentry\Integration\CycleOrm;

use Cycle\ORM\Select\Repository;
use Modules\Sentry\Domain\Issue;
use Modules\Sentry\Domain\IssueRepositoryInterface;

final class IssueRepository extends Repository implements IssueRepositoryInterface
{

    public function findLatestByFingerprint(string $fingerprint): ?Issue
    {
        return $this->select()
            ->where('fingerprint.fingerprint', $fingerprint)
            ->orderBy('created_at', 'DESC')
            ->fetchOne();
    }
}
