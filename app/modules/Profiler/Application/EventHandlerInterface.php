<?php

declare(strict_types=1);

namespace Modules\Profiler\Application;

/**
 * @psalm-type TEvent = array{
 *     profile: array<non-empty-string, array{ct: int, wt: int, cpu: int, mu: int, pmu: int}>,
 *     tags: array<non-empty-string, string|int>,
 *     app_name: non-empty-string,
 *     hostname: non-empty-string,
 *     date: int
 * }
 */
interface EventHandlerInterface
{
    /**
     * @param TEvent $event
     * @return TEvent
     */
    public function handle(array $event): array;
}
