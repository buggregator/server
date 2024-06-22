<?php

declare(strict_types=1);

namespace Modules\Sentry\Domain;

use Cycle\ORM\RepositoryInterface;

interface IssueRepositoryInterface extends RepositoryInterface
{
    public function findLatestByFingerprint(string $fingerprint): ?Issue;
}
