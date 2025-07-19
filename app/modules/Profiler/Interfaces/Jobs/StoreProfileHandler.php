<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Jobs;

use Modules\Profiler\Domain\Edge\Cost;
use Modules\Profiler\Domain\Edge\Diff;
use Modules\Profiler\Domain\Edge\Percents;
use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\ORMInterface;
use Modules\Profiler\Application\MetricsHelper;
use Modules\Profiler\Application\Query\FindTopFunctionsByUuid;
use Modules\Profiler\Domain\EdgeFactoryInterface;
use Modules\Profiler\Domain\Profile;
use Spiral\Core\InvokerInterface;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Queue\JobHandler;

// TODO: refactor this, use repository
final class StoreProfileHandler extends JobHandler
{
    private const BATCH_SIZE = 100;

    public function __construct(
        private readonly EdgeFactoryInterface $edgeFactory,
        private readonly EntityManagerInterface $em,
        private readonly QueryBusInterface $bus,
        private readonly ORMInterface $orm,
        InvokerInterface $invoker,
    ) {
        parent::__construct($invoker);
    }

    public function invoke(array $payload): void
    {
        $event = $payload;
        $profileUuid = Uuid::fromString($event['profile_uuid']);
        $parents = [];

        $edges = &$event['edges'];

        if (!\array_key_exists('main()', $edges) && \array_key_exists('value', $edges)) {
            $edges['main()'] = $edges['value'];
        }
        unset($edges['value']);

        $batchSize = 0;
        $i = 0;
        foreach ($edges as $id => $edge) {
            // Use safe metric access with defaults for missing values
            $cost = $edge['cost'] ?? [];
            $normalizedCost = MetricsHelper::getAllMetrics($cost);

            $this->em->persist(
                $edge = $this->edgeFactory->create(
                    profileUuid: $profileUuid,
                    order: $i++,
                    cost: new Cost(
                        cpu: $normalizedCost['cpu'],
                        wt: $normalizedCost['wt'],
                        ct: $normalizedCost['ct'],
                        mu: $normalizedCost['mu'],
                        pmu: $normalizedCost['pmu'],
                    ),
                    diff: new Diff(
                        cpu: MetricsHelper::getMetric($cost, 'd_cpu'),
                        wt: MetricsHelper::getMetric($cost, 'd_wt'),
                        ct: MetricsHelper::getMetric($cost, 'd_ct'),
                        mu: MetricsHelper::getMetric($cost, 'd_mu'),
                        pmu: MetricsHelper::getMetric($cost, 'd_pmu'),
                    ),
                    percents: new Percents(
                        cpu: (float) MetricsHelper::getMetric($cost, 'p_cpu'),
                        wt: (float) MetricsHelper::getMetric($cost, 'p_wt'),
                        ct: (float) MetricsHelper::getMetric($cost, 'p_ct'),
                        mu: (float) MetricsHelper::getMetric($cost, 'p_mu'),
                        pmu: (float) MetricsHelper::getMetric($cost, 'p_pmu'),
                    ),
                    callee: $edge['callee'],
                    caller: $edge['caller'],
                    parentUuid: $edge['parent'] ? $parents[$edge['parent']] ?? null : null,
                ),
            );

            $parents[$id] = $edge->getUuid();

            if (self::BATCH_SIZE === $batchSize) {
                $this->em->run();
                $batchSize = 0;
            }

            $batchSize++;
        }

        $profile = $this->orm->getRepository(Profile::class)->findByPK($profileUuid);
        $functions = $this->bus->ask(new FindTopFunctionsByUuid(profileUuid: $profileUuid));

        // Safely update peaks with normalized metrics
        foreach ($functions['overall_totals'] as $metric => $value) {
            if (property_exists($profile->getPeaks(), $metric)) {
                $profile->getPeaks()->{$metric} = $value;
            }
        }

        $this->em->persist($profile);
        $this->em->run();
    }
}
