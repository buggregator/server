<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Jobs;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\EntityManagerInterface;
use Modules\Profiler\Domain\Edge\Cost;
use Modules\Profiler\Domain\Edge\Diff;
use Modules\Profiler\Domain\Edge\Percents;
use Modules\Profiler\Domain\EdgeFactoryInterface;
use Modules\Profiler\Domain\Profile\Peaks;
use Modules\Profiler\Domain\ProfileFactoryInterface;
use Spiral\Core\InvokerInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Queue\JobHandler;

/**
 * Handles pre-processed profile events from the RoadRunner Profiler plugin.
 * The Go plugin already computed peaks, diffs, edges tree, and percentages —
 * this handler only persists them to the database and dispatches the event.
 */
final class ProfileHandler extends JobHandler
{
    private const BATCH_SIZE = 100;

    public function __construct(
        private readonly ProfileFactoryInterface $profileFactory,
        private readonly EdgeFactoryInterface $edgeFactory,
        private readonly EntityManagerInterface $em,
        private readonly CommandBusInterface $bus,
        InvokerInterface $invoker,
    ) {
        parent::__construct($invoker);
    }

    public function invoke(mixed $payload): void
    {
        if (\is_string($payload)) {
            $payload = \json_decode($payload, true) ?? [];
        }

        $profileUuid = Uuid::generate();

        // Create Profile entity with peaks from Go plugin
        $peaks = $payload['peaks'] ?? [];
        $profile = $this->profileFactory->create(
            name: $payload['app_name'] ?? 'unknown',
            tags: $payload['tags'] ?? [],
            peaks: new Peaks(
                cpu: $peaks['cpu'] ?? 0,
                wt: $peaks['wt'] ?? 0,
                ct: $peaks['ct'] ?? 0,
                mu: $peaks['mu'] ?? 0,
                pmu: $peaks['pmu'] ?? 0,
            ),
        );

        $this->em->persist($profile)->run();

        // Store pre-processed edges
        $edges = $payload['edges'] ?? [];
        $parents = [];
        $batchSize = 0;
        $order = 0;

        foreach ($edges as $id => $edge) {
            $cost = $edge['cost'] ?? [];
            $diff = $edge['diff'] ?? [];
            $pcts = $edge['percents'] ?? [];

            $edgeEntity = $this->edgeFactory->create(
                profileUuid: $profile->getUuid(),
                order: $order++,
                cost: new Cost(
                    cpu: $cost['cpu'] ?? 0,
                    wt: $cost['wt'] ?? 0,
                    ct: $cost['ct'] ?? 0,
                    mu: $cost['mu'] ?? 0,
                    pmu: $cost['pmu'] ?? 0,
                ),
                diff: new Diff(
                    cpu: $diff['d_cpu'] ?? 0,
                    wt: $diff['d_wt'] ?? 0,
                    ct: $diff['d_ct'] ?? 0,
                    mu: $diff['d_mu'] ?? 0,
                    pmu: $diff['d_pmu'] ?? 0,
                ),
                percents: new Percents(
                    cpu: $pcts['p_cpu'] ?? 0.0,
                    wt: $pcts['p_wt'] ?? 0.0,
                    ct: $pcts['p_ct'] ?? 0.0,
                    mu: $pcts['p_mu'] ?? 0.0,
                    pmu: $pcts['p_pmu'] ?? 0.0,
                ),
                callee: $edge['callee'],
                caller: $edge['caller'] ?? null,
                parentUuid: isset($edge['parent']) ? ($parents[$edge['parent']] ?? null) : null,
            );

            $this->em->persist($edgeEntity);
            $parents[$id] = $edgeEntity->getUuid();

            $batchSize++;
            if ($batchSize >= self::BATCH_SIZE) {
                $this->em->run();
                $batchSize = 0;
            }
        }

        $this->em->run();

        // Dispatch event for broadcasting and storage in events table
        $this->bus->dispatch(
            new HandleReceivedEvent(
                type: 'profiler',
                payload: [
                    'profile_uuid' => (string) $profile->getUuid(),
                    'peaks' => $peaks,
                    'tags' => $payload['tags'] ?? [],
                    'app_name' => $payload['app_name'] ?? 'unknown',
                    'hostname' => $payload['hostname'] ?? 'unknown',
                    'date' => $payload['date'] ?? 0,
                    'total_edges' => \count($edges),
                ],
                uuid: $profileUuid,
            ),
        );
    }
}
