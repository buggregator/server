<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Queries;

use Cycle\ORM\ORMInterface;
use Modules\Profiler\Application\Query\FindFlameChartByUuid;
use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\Profile;
use Spiral\Cqrs\Attribute\QueryHandler;
use Spiral\Storage\BucketInterface;

// TODO: refactor this, use repository
final readonly class FindFlameChartByUuidHandler
{
    public function __construct(
        private ORMInterface $orm,
        private BucketInterface $bucket,
    ) {}

    #[QueryHandler]
    public function __invoke(FindFlameChartByUuid $query): string
    {
        $metric = $query->metric->value;
        $file = $query->profileUuid . '.' . $metric . '.flamechart.json';
        if ($this->bucket->exists($file)) {
            return $this->bucket->getContents($file);
        }

        $profile = $this->orm->getRepository(Profile::class)->findByPK($query->profileUuid);

        /** @var Edge[] $edges */
        $edges = $profile->edges;

        $waterfall = [];
        $eventCache = [];

        // Divisor: microseconds → milliseconds for time metrics, bytes → KB for memory
        $divisor = match ($metric) {
            'mu', 'pmu' => 1_024,
            default => 1_000,
        };

        foreach ($edges as $edge) {
            $rawValue = $edge->getCost()->{$metric} ?? 0;
            $eventData = [
                'name' => $edge->getCallee(),
                'start' => 0,
                'duration' => $rawValue > 0 ? \round($rawValue / $divisor, 3) : 0,
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

        $this->adjustStartTimes($waterfall);
        $serialized = \json_encode($waterfall, 0, 5000);
        $this->bucket->write($file, $serialized);

        return $serialized;
    }

    /**
     * Lay out children sequentially within their parent's time span.
     * If children's total duration exceeds the parent, scale them down proportionally.
     */
    private function adjustStartTimes(array &$eventList, float|int $startTime = 0, float|int $parentDuration = 0): void
    {
        $childrenTotal = 0;
        foreach ($eventList as &$event) {
            $childrenTotal += $event['duration'];
        }
        unset($event);

        // Scale factor: if children overflow parent, shrink proportionally
        $scale = ($parentDuration > 0 && $childrenTotal > $parentDuration)
            ? $parentDuration / $childrenTotal
            : 1.0;

        $cursor = $startTime;
        foreach ($eventList as &$event) {
            $event['start'] = \round($cursor, 3);
            $scaledDuration = \round($event['duration'] * $scale, 3);

            // Recursively adjust children within this event's bounds
            $this->adjustStartTimes($event['children'], $event['start'], $scaledDuration);

            $cursor += $scaledDuration;
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
