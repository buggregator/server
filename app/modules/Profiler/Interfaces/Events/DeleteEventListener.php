<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Events;

use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\ORMInterface;
use Modules\Events\Domain\Events\EventWasDeleted;
use Modules\Profiler\Domain\Profile;
use Spiral\Events\Attribute\Listener;

final readonly class DeleteEventListener
{
    public function __construct(
        private ORMInterface $orm,
        private EntityManagerInterface $em,
    ) {}

    #[Listener]
    public function __invoke(EventWasDeleted $event): void
    {
        $profile = $this->orm->getRepository(Profile::class)->findByPK($event->uuid);
        if ($profile === null) {
            return;
        }

        $this->em->delete($profile)->run();
    }
}
