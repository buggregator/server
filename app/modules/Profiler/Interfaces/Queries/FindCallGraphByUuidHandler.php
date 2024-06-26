<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Queries;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\ORMInterface;
use Modules\Profiler\Application\CallGraph\Node;
use Modules\Profiler\Application\Query\FindCallGraphByUuid;
use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\Profile;
use Spiral\Cqrs\Attribute\QueryHandler;

// TODO: refactor this, use repository
final readonly class FindCallGraphByUuidHandler
{
    public function __construct(
        private ORMInterface $orm,
    ) {}

    #[QueryHandler]
    public function __invoke(FindCallGraphByUuid $query): array
    {
        $profile = $this->orm->getRepository(Profile::class)->findByPK($query->profileUuid);

        $edges = $profile->edges;
        $registered = [];

        return \array_reduce(
            $edges->toArray(),
            static function (array $carry, Edge $edge) use (&$registered, $query): array {
                $node = Node::fromEdge(
                    edge: $edge,
                    metric: $query->metric,
                    maxColorPercentage: $query->percentage,
                );

                if (!$node->isImportant() && $node->isSatisfied($query->threshold)) {
                    return $carry;
                }

                $carry['nodes'][] = ['data' => $node];

                $registered[] = (string) $edge->getUuid();

                if ($edge->getParentUuid() instanceof Uuid && \in_array((string) $edge->getParentUuid(), $registered, true)) {
                    $carry['edges'][] = [
                        'data' => [
                            'source' => (string) $edge->getParentUuid(),
                            'target' => (string) $edge->getUuid(),
                            'label' => \sprintf(
                                '%s%%',
                                $node->getPercentsMetric(),
                            ),
                            'color' => $node->color,
                        ],
                    ];
                }

                return $carry;
            },
            [
                'toolbar' => [
                    [
                        'label' => 'CPU',
                        'metric' => 'cpu',
                        'description' => 'CPU time in ms.',
                    ],
                    [
                        'label' => 'Wall time',
                        'metric' => 'wt',
                        'description' => 'Wall time in ms.',
                    ],
                    [
                        'label' => 'Memory',
                        'metric' => 'mu',
                        'description' => 'Memory usage in bytes.',
                    ],
                    [
                        'label' => 'Memory peak',
                        'metric' => 'pmu',
                        'description' => 'Memory peak usage in bytes.',
                    ],
                ],
                'nodes' => [],
                'edges' => [],
            ],
        );
    }
}
