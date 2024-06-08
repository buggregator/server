<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Handlers;

use Cycle\ORM\EntityManagerInterface;
use Modules\Profiler\Application\EventHandlerInterface;
use Modules\Profiler\Domain\Profile\Peaks;
use Modules\Profiler\Domain\ProfileFactoryInterface;
use Modules\Profiler\Interfaces\Jobs\StoreProfileHandler;
use Spiral\Queue\QueueInterface;

// TODO: refactor this, use repository
final readonly class StoreProfile implements EventHandlerInterface
{
    public function __construct(
        private ProfileFactoryInterface $profileFactory,
        private EntityManagerInterface $em,
        private QueueInterface $queue,
    ) {}

    public function handle(array $event): array
    {
        $profile = $this->profileFactory->create(
            name: $event['app_name'],
            tags: $event['tags'],
            peaks: new Peaks(
                cpu: $event['peaks']['cpu'] ?? 0,
                wt: $event['peaks']['wt'] ?? 0,
                ct: $event['peaks']['ct'] ?? 0,
                mu: $event['peaks']['mu'] ?? 0,
                pmu: $event['peaks']['pmu'] ?? 0,
            ),
        );

        $this->em->persist($profile)->run();
        $event['profile_uuid'] = (string) $profile->getUuid();

        $this->queue->push(StoreProfileHandler::class, $event);

        unset($event['edges']);

        return $event;
    }
}
