<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Events;

use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\ORMInterface;
use Modules\Events\Domain\Events\EventWasDeleted;
use Modules\Profiler\Application\CallGraph\Metric;
use Modules\Profiler\Domain\Profile;
use Spiral\Events\Attribute\Listener;
use Spiral\Storage\BucketInterface;

final readonly class DeleteEventListener
{
    public function __construct(
        private ORMInterface $orm,
        private EntityManagerInterface $em,
        private BucketInterface $bucket,
    ) {}

    #[Listener]
    public function __invoke(EventWasDeleted $event): void
    {
        $profile = $this->orm->getRepository(Profile::class)->findByPK($event->uuid);
        if (!$profile) {
            return;
        }

        // Clean up cached flame chart files for all metrics
        foreach (Metric::cases() as $metric) {
            $file = $event->uuid . '.' . $metric->value . '.flamechart.json';
            if ($this->bucket->exists($file)) {
                $this->bucket->delete($file);
            }
        }

        $this->em->delete($profile)->run();
    }
}
