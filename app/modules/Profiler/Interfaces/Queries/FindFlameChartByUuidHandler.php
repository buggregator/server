<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Queries;

use Modules\Profiler\Application\Query\FindFlameChartByUuid;
use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\ProfileRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;
use Spiral\Storage\BucketInterface;

final readonly class FindFlameChartByUuidHandler
{
    public function __construct(
        private ProfileRepositoryInterface $profiles,
        private BucketInterface $bucket,
    ) {}

    #[QueryHandler]
    public function __invoke(FindFlameChartByUuid $query): array
    {
        $file = $query->profileUuid . '.flamechart.json';
        if ($this->bucket->exists($file)) {
            return \json_decode($this->bucket->getContents($file), true);
        }

        $profile = $this->profiles->getByUuid($query->profileUuid);

        /** @var Edge[] $edges */
        $edges = $profile->edges;

        $waterfall = [];
        $eventCache = [];
        // TODO: send metric from the frontend side
        $metric = 'wt';

        foreach ($edges as $edge) {
            $duration = $edge->getCost()->{$metric} ?? 0;
            $eventData = [
                'name' => $edge->getCallee(),
                'start' => 0,  // Temporarily zero, will adjust based on the parent later
                'duration' => $duration > 0 ? \round($duration / 1_000, 3) : 0,
                'type' => 'task',
                'children' => [],
                'cost' => [
                    'cpu' => $edge->getCost()->cpu,
                    'wt' => $edge->getCost()->wt,
                    'pmu' => $edge->getCost()->pmu,
                    'mu' => $edge->getCost()->mu,
                    'ct' => $edge->getCost()->ct,
                ],
                'color' => $this->getColorForPercentCount($edge->getPercents()->{$metric} ?? 0),
            ];

            $id = (string) $edge->getUuid();
            $parent = $edge->getParentUuid() ? (string) $edge->getParentUuid() : null;
            $eventCache[$id] = $eventData;

            if ($parent && isset($eventCache[$parent])) {
                $eventCache[$parent]['children'][] = &$eventCache[$id];
            } else {
                $waterfall[] = &$eventCache[$id];
            }
        }

        $this->adjustStartTimes($waterfall, 0);
        $this->bucket->write($file, \json_encode($waterfall, 0, 5000));

        return $waterfall;
    }

    private function adjustStartTimes(array &$eventList, float|int $startTime): void
    {
        foreach ($eventList as &$event) {
            $event['start'] = $startTime;
            // Next event starts after the current event ends.
            $startTime += $event['duration'];
            // Recursively adjust times for children.
            $this->adjustStartTimes($event['children'], $event['start']);
        }
    }

    private function getColorForPercentCount(float|int $percents): string
    {
        return match (true) {
            $percents <= 10 => '#B3E5FC', // Light Blue
            $percents <= 20 => '#81D4FA', // Light Sky Blue
            $percents <= 30 => '#4FC3F7', // Vivid Light Blue
            $percents <= 40 => '#29B6F6', // Bright Light Blue
            $percents <= 50 => '#FFCDD2', // Pink (Light Red)
            $percents <= 60 => '#FFB2B2', // Lighter Red
            $percents <= 70 => '#FF9E9E', // Soft Red
            $percents <= 80 => '#FF8989', // Soft Coral
            $percents <= 90 => '#FF7474', // Soft Tomato
            default => '#FF5F5F', // Light Coral
        };
    }

}
