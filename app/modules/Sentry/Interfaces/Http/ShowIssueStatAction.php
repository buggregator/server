<?php

declare(strict_types=1);

namespace Modules\Sentry\Interfaces\Http;

use Modules\Sentry\Domain\FingerprintRepositoryInterface;
use Spiral\Router\Annotation\Route;

final readonly class ShowIssueStatAction
{
    public function __construct(
        private FingerprintRepositoryInterface $fingerprints,
    ) {}

    #[Route(route: '/sentry/issue/<fingerprint>/stat', name: 'sentry.issue.stat', methods: 'GET', group: 'api')]
    public function __invoke(string $fingerprint): array
    {
        $days = 14;

        $stat = $this->fingerprints->stat($fingerprint, $days);
        $firstEvent = null;
        $lastEvent = null;
        $totalEvents = $this->fingerprints->totalEvents($fingerprint);

        if ($totalEvents === 1) {
            $firstEvent = $lastEvent = $this->fingerprints->findFirstSeen($fingerprint);
        } elseif ($totalEvents > 1) {
            $firstEvent = $this->fingerprints->findFirstSeen($fingerprint);
            $lastEvent = $this->fingerprints->findLastSeen($fingerprint);
        }

        return [
            'total_events' => $totalEvents,
            'first_event' => $firstEvent?->getCreatedAt()?->format(\DateTimeInterface::W3C),
            'last_event' => $lastEvent?->getCreatedAt()?->format(\DateTimeInterface::W3C),
            'fingerprint' => $fingerprint,
            'stat_days' => $days,
            'stat' => $stat,
        ];
    }
}
