<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\CallGraph;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Profiler\Domain\Edge;

final readonly class Node implements \JsonSerializable
{
    public static function fromEdge(
        Edge $edge,
        Metric $metric,
        int $maxColorPercentage,
    ): self {
        return new self(
            $edge->getUuid(),
            $edge->getCallee(),
            $metric,
            $edge->getCost(),
            $edge->getPercents(),
            $maxColorPercentage,
        );
    }

    public string $color;
    public string $textColor;
    public string $label;

    public function __construct(
        public Uuid $uuid,
        public string $callee,
        public Metric $metric,
        public Edge\Cost $cost,
        public Edge\Percents $percents,
        public int $maxColorPercentage,
    ) {
        $this->color = $this->isImportant() ? $this->detectNodeColor() : '#FFFFFF';
        $this->textColor = $this->isImportant() ? $this->detectTextColor($this->color) : '#000000';
        $this->label = $this->buildLabel();
    }

    public function isImportant(): bool
    {
        return $this->getPercentsMetric() >= $this->maxColorPercentage;
    }

    public function isSatisfied(int|float $threshold): bool
    {
        return $this->getPercentsMetric() <= $threshold;
    }

    public function getCostMetric(): float|int
    {
        return $this->cost->{$this->metric->value};
    }

    public function getPercentsMetric(): float|int
    {
        return $this->percents->{$this->metric->value};
    }

    public function jsonSerialize(): array
    {
        return [
            'id' => (string) $this->uuid,
            'name' => $this->label,
            'cost' => [
                'cpu' => $this->cost->cpu,
                'wt' => $this->cost->wt,
                'pmu' => $this->cost->pmu,
                'mu' => $this->cost->mu,
                'ct' => $this->cost->ct,
            ],
            'metrics' => [
                'cost' => $this->getCostMetric(),
                'percents' => $this->getPercentsMetric(),
            ],
            'color' => $this->color,
            'textColor' => $this->textColor,
        ];
    }

    private function buildLabel(): string
    {
        if ($this->cost->ct > 0) {
            return \sprintf(
                '%s (%sx)',
                $this->callee,
                $this->cost->ct,
            );
        }

        return $this->callee;
    }

    private function detectNodeColor(): string
    {
        $percent = $this->getPercentsMetric();

        return match (true) {
            $percent <= 10 => '#FFFFFF', // White
            $percent <= 20 => '#f19797', // Lighter shade towards dark red
            $percent <= 30 => '#d93939', // Light shade towards dark red
            $percent <= 40 => '#ad1e1e', // Intermediate lighter shade towards dark red
            $percent <= 50 => '#982525', // Intermediate shade towards dark red
            $percent <= 60 => '#862323', // Intermediate darker shade towards dark red
            $percent <= 70 => '#671d1d', // Darker shade towards dark red
            $percent <= 80 => '#540d0d', // More towards dark red
            $percent <= 90 => '#340707', // Almost dark red
            default => '#000000', // Black
        };
    }

    private function detectTextColor(string $nodeColor): string
    {
        $hex = \ltrim($nodeColor, '#');
        $r = \hexdec(\substr($hex, 0, 2));
        $g = \hexdec(\substr($hex, 2, 2));
        $b = \hexdec(\substr($hex, 4, 2));
        $brightness = ($r * 299 + $g * 587 + $b * 114) / 1000;

        return $brightness > 125 ? '#000000' : '#FFFFFF';
    }
}
