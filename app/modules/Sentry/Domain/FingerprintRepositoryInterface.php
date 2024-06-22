<?php

declare(strict_types=1);

namespace Modules\Sentry\Domain;

use Cycle\ORM\RepositoryInterface;

interface FingerprintRepositoryInterface extends RepositoryInterface
{
    public function findFirstSeen(string $fingerprint): ?Fingerprint;

    public function findLastSeen(string $fingerprint): ?Fingerprint;

    public function totalEvents(string $fingerprint): int;

    public function stat(string $fingerprint, int $days = 7): array;
}
