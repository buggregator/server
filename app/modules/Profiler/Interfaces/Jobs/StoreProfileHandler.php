<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Jobs;

use Modules\Profiler\Domain\Edge\Cost;
use Modules\Profiler\Domain\Edge\Diff;
use Modules\Profiler\Domain\Edge\Percents;
use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\ORMInterface;
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

        $batchSize = 0;
        $i = 0;
        foreach ($event['edges'] as $id => $edge) {
            $this->em->persist(
                $edge = $this->edgeFactory->create(
                    profileUuid: $profileUuid,
                    order: $i++,
                    cost: new Cost(
                        cpu: $edge['cost']['cpu'] ?? 0,
                        wt: $edge['cost']['wt'] ?? 0,
                        ct: $edge['cost']['ct'] ?? 0,
                        mu: $edge['cost']['mu'] ?? 0,
                        pmu: $edge['cost']['pmu'] ?? 0,
                    ),
                    diff: new Diff(
                        cpu: $edge['cost']['d_cpu'] ?? 0,
                        wt: $edge['cost']['d_wt'] ?? 0,
                        ct: $edge['cost']['d_ct'] ?? 0,
                        mu: $edge['cost']['d_mu'] ?? 0,
                        pmu: $edge['cost']['d_pmu'] ?? 0,
                    ),
                    percents: new Percents(
                        cpu: $edge['cost']['p_cpu'] ?? 0,
                        wt: $edge['cost']['p_wt'] ?? 0,
                        ct: $edge['cost']['p_ct'] ?? 0,
                        mu: $edge['cost']['p_mu'] ?? 0,
                        pmu: $edge['cost']['p_pmu'] ?? 0,
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

            ++$batchSize;
        }

        $profile = $this->orm->getRepository(Profile::class)->findByPK($profileUuid);
        $functions = $this->bus->ask(new FindTopFunctionsByUuid(profileUuid: $profileUuid));

        foreach ($functions['overall_totals'] as $metric => $value) {
            $profile->getPeaks()->{$metric} = $value;
        }

        $this->em->persist($profile);
        $this->em->run();
    }
}
