<?php

declare(strict_types=1);

namespace Modules\Sentry\Integration\CycleOrm;

use Carbon\Carbon;
use Carbon\CarbonInterval;
use Carbon\CarbonPeriod;
use Cycle\Database\DatabaseInterface;
use Cycle\Database\Injection\Fragment;
use Cycle\ORM\Select;
use Cycle\ORM\Select\Repository;
use Modules\Sentry\Domain\Fingerprint;
use Modules\Sentry\Domain\FingerprintRepositoryInterface;

final class FingerprintRepository extends Repository implements FingerprintRepositoryInterface
{
    public function __construct(
        Select $select,
        private DatabaseInterface $database,
    ) {
        parent::__construct($select);
    }

    public function findFirstSeen(string $fingerprint): ?Fingerprint
    {
        return $this->select()
            ->where('fingerprint', $fingerprint)
            ->orderBy('created_at', 'ASC')
            ->fetchOne();
    }

    public function findLastSeen(string $fingerprint): ?Fingerprint
    {
        return $this->select()
            ->where('fingerprint', $fingerprint)
            ->orderBy('created_at', 'DESC')
            ->fetchOne();
    }

    public function totalEvents(string $fingerprint): int
    {
        return $this->select()
            ->where('fingerprint', $fingerprint)
            ->count();
    }

    public function stat(string $fingerprint, int $days = 7): array
    {
        $rage = Carbon::now()->subDays($days)->toPeriod(Carbon::now(), CarbonInterval::day());

        $result = $this->database->select([
            new Fragment('DATE(created_at) as date'),
            new Fragment('COUNT(*) as count'),
        ])
            ->from('sentry_issue_fingerprints')
            ->where('created_at', '>=', $rage->getStartDate())
            ->where('fingerprint', $fingerprint)
            ->groupBy('date')
            ->fetchAll();

        $result = array_combine(
            array_column($result, 'date'),
            array_column($result, 'count'),
        );

        $stat = [];
        foreach ($rage as $date) {
            $stat[$date->format('Y-m-d')] = [
                'date' => $date->format('Y-m-d'),
                'count' => $result[$date->format('Y-m-d')] ?? 0,
            ];
        }

        return \array_values($stat);
    }
}
