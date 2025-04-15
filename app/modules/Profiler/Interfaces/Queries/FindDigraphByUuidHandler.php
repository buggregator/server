<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Queries;

use Modules\Profiler\Application\Query\FindDigraphByUuid;
use Modules\Profiler\Application\Query\FindTopFunctionsByUuid;
use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\ProfileRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;
use Spiral\Cqrs\QueryBusInterface;

final readonly class FindDigraphByUuidHandler
{
    public function __construct(
        private ProfileRepositoryInterface $profiles,
        private QueryBusInterface $bus,
    ) {}

    #[QueryHandler]
    public function __invoke(FindDigraphByUuid $query): string
    {
        $profile = $this->profiles->getByUuid($query->profileUuid);
        $topFn = $this->bus->ask(new FindTopFunctionsByUuid($query->profileUuid))['functions'];
        $sym_table = [];

        foreach ($topFn as $fn) {
            $sym_table[$fn['function']] = $fn;
        }

        unset($topFn);

        $totalWt = $profile->getPeaks()->wt;

        $right = null;
        $left = null;

        $maxWidth = 5;
        $maxHeight = 3.5;
        $maxFontSize = 35;
        $maxSizingRatio = 20;
        $source = 'xhprof';
        $threshold = $query->threshold;

        /** @var array<string, Edge> $edges */
        $edges = [];

        foreach ($profile->edges as $edge) {
            $edges[$this->buildKey($edge->getCaller(), $edge->getCallee())] = $edge;
        }

        if ($query->criticalPath) {
            $tree = $this->buildTree($edges);

            $node = "main()";
            $path = [];
            $pathEdges = [];
            $visited = [];
            while ($node) {
                $visited[$node] = true;
                if (isset($tree[$node])) {
                    $maxChild = null;
                    foreach ($tree[$node] as $child) {
                        if (isset($visited[$child])) {
                            continue;
                        }
                        if ($maxChild === null ||
                            \abs($edges[$this->buildKey($node, $child)]->getCost()->wt) >
                            \abs($edges[$this->buildKey($node, $maxChild)]->getCost()->wt)
                        ) {
                            $maxChild = $child;
                        }
                    }
                    if ($maxChild !== null) {
                        $path[$maxChild] = true;
                        $pathEdges[$this->buildKey($node, $maxChild)] = true;
                    }
                    $node = $maxChild;
                } else {
                    $node = null;
                }
            }
        }

        $result = "digraph call_graph {\n";

        $cur_id = 0;
        $max_wt = 0;

        foreach ($sym_table as $symbol => $info) {
            $fn = $info["function"];
            if (\abs($info["wt"] / $totalWt) < $threshold) {
                unset($sym_table[$symbol]);
                continue;
            }
            if ($max_wt == 0 || $max_wt < \abs($info["excl_wt"])) {
                $max_wt = \abs($info["excl_wt"]);
            }
            $sym_table[$symbol]["id"] = $cur_id;
            $cur_id++;
        }

        // Generate all nodes' information.
        foreach ($sym_table as $symbol => $info) {
            $fn = $info["function"];
            if ($info["excl_wt"] == 0) {
                $sizingFactor = $maxSizingRatio;
            } else {
                $sizingFactor = $max_wt / abs($info["excl_wt"]);
                if ($sizingFactor > $maxSizingRatio) {
                    $sizingFactor = $maxSizingRatio;
                }
            }
            $fillcolor = (($sizingFactor < 1.5) ?
                ", style=filled, fillcolor=red" : "");

            if ($query->criticalPath) {
                // highlight nodes along critical path.
                if (!$fillcolor && array_key_exists($symbol, $path)) {
                    $fillcolor = ", style=filled, fillcolor=yellow";
                }
            }

            $fontsize = ", fontsize="
                . (int) ($maxFontSize / (($sizingFactor - 1) / 10 + 1));

            $width = ", width=" . sprintf("%.1f", $maxWidth / $sizingFactor);
            $height = ", height=" . sprintf("%.1f", $maxHeight / $sizingFactor);

            if ($symbol == "main()") {
                $shape = "octagon";
                $name = "Total: " . ($totalWt / 1000.0) . " ms\\n";
                $name .= addslashes($fn);
            } else {
                $shape = "box";
                $name = addslashes($fn) . "\\nInc: " . sprintf("%.3f", $info["wt"] / 1000) .
                    " ms (" . sprintf("%.1f%%", 100 * $info["wt"] / $totalWt) . ")";
            }
            if ($left === null) {
                $label = ", label=\"" . $name . "\\nExcl: "
                    . (sprintf("%.3f", $info["excl_wt"] / 1000.0)) . " ms ("
                    . sprintf("%.1f%%", 100 * $info["excl_wt"] / $totalWt)
                    . ")\\n" . $info["ct"] . " total calls\"";
            } else {
                if (isset($left[$symbol]) && isset($right[$symbol])) {
                    $label = ", label=\"" . addslashes($fn) .
                        "\\nInc: " . (sprintf("%.3f", $left[$symbol]["wt"] / 1000.0))
                        . " ms - "
                        . (sprintf("%.3f", $right[$symbol]["wt"] / 1000.0)) . " ms = "
                        . (sprintf("%.3f", $info["wt"] / 1000.0)) . " ms" .
                        "\\nExcl: "
                        . (sprintf("%.3f", $left[$symbol]["excl_wt"] / 1000.0))
                        . " ms - " . (sprintf("%.3f", $right[$symbol]["excl_wt"] / 1000.0))
                        . " ms = " . (sprintf("%.3f", $info["excl_wt"] / 1000.0)) . " ms" .
                        "\\nCalls: " . (sprintf("%.3f", $left[$symbol]["ct"])) . " - "
                        . (sprintf("%.3f", $right[$symbol]["ct"])) . " = "
                        . (sprintf("%.3f", $info["ct"])) . "\"";
                } else {
                    if (isset($left[$symbol])) {
                        $label = ", label=\"" . addslashes($fn) .
                            "\\nInc: " . (sprintf("%.3f", $left[$symbol]["wt"] / 1000.0))
                            . " ms - 0 ms = " . (sprintf("%.3f", $info["wt"] / 1000.0))
                            . " ms" . "\\nExcl: "
                            . (sprintf("%.3f", $left[$symbol]["excl_wt"] / 1000.0))
                            . " ms - 0 ms = "
                            . (sprintf("%.3f", $info["excl_wt"] / 1000.0)) . " ms" .
                            "\\nCalls: " . (sprintf("%.3f", $left[$symbol]["ct"])) . " - 0 = "
                            . (sprintf("%.3f", $info["ct"])) . "\"";
                    } else {
                        $label = ", label=\"" . addslashes($fn) .
                            "\\nInc: 0 ms - "
                            . (sprintf("%.3f", $right[$symbol]["wt"] / 1000.0))
                            . " ms = " . (sprintf("%.3f", $info["wt"] / 1000.0)) . " ms" .
                            "\\nExcl: 0 ms - "
                            . (sprintf("%.3f", $right[$symbol]["excl_wt"] / 1000.0))
                            . " ms = " . (sprintf("%.3f", $info["excl_wt"] / 1000.0)) . " ms" .
                            "\\nCalls: 0 - " . (sprintf("%.3f", $right[$symbol]["ct"]))
                            . " = " . (sprintf("%.3f", $info["ct"])) . "\"";
                    }
                }
            }
            $result .= "N" . $sym_table[$symbol]["id"];
            $result .= "[shape=$shape " . $label . $width
                . $height . $fontsize . $fillcolor . "];\n";
        }


        // Generate all the edges' information.
        foreach ($edges as $edge) {
            [$parent, $child] = [$edge->getCaller(), $edge->getCallee()];

            if (isset($sym_table[$parent]) && isset($sym_table[$child])) {
                $label = $edge->getCost()->ct == 1 ? $edge->getCost()->ct . " call" : $edge->getCost()->ct . " calls";

                $headlabel = $sym_table[$child]["wt"] > 0 ?
                    sprintf(
                        "%.1f%%",
                        100 * $info["wt"]
                        / $sym_table[$child]["wt"],
                    )
                    : "0.0%";

                $taillabel = ($sym_table[$parent]["wt"] > 0) ?
                    sprintf(
                        "%.1f%%",
                        100 * $info["wt"] /
                        ($sym_table[$parent]["wt"] - $sym_table["$parent"]["excl_wt"]),
                    )
                    : "0.0%";

                $linewidth = 1;
                $arrow_size = 1;

                if ($query->criticalPath && isset($tree[$this->buildKey($parent, $child)])) {
                    $linewidth = 10;
                    $arrow_size = 2;
                }

                $result .= "N" . $sym_table[$parent]["id"] . " -> N"
                    . $sym_table[$child]["id"];
                $result .= "[arrowsize=$arrow_size, color=grey, style=\"setlinewidth($linewidth)\","
                    . " label=\""
                    . $label . "\", headlabel=\"" . $headlabel
                    . "\", taillabel=\"" . $taillabel . "\" ]";
                $result .= ";\n";
            }
        }

        return $result . "\n}";
    }

    /**
     * @param array<string, Edge> $edges
     */
    private function buildTree(array $edges): array
    {
        $tree = [];

        foreach ($edges as $edge) {
            $parent = $edge->getCaller();
            $child = $edge->getCallee();
            if (!isset($tree[$parent])) {
                $tree[$parent] = [$child];
            } else {
                $tree[$parent][] = $child;
            }
        }

        return $tree;
    }

    private function buildKey(?string $parent, ?string $child): string
    {
        return $parent . '==>' . $child;
    }
}
