<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use Cycle\Annotated\Annotation\Relation\BelongsTo;
use Cycle\Annotated\Annotation\Relation\Embedded;
use Modules\Profiler\Domain\Edge\Cost;
use Modules\Profiler\Domain\Edge\Diff;
use Modules\Profiler\Domain\Edge\Percents;

// TODO: add repository
#[Entity(
    role: 'profile_edge',
    table: 'profile_edges',
)]
class Edge
{
    #[BelongsTo(target: Edge::class, innerKey: 'parent_uuid', outerKey: 'uuid', nullable: true)]
    private ?Edge $parent = null;

    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: 'uuid')]
        private Uuid $uuid,
        #[Column(type: 'string(36)', name: 'profile_uuid', typecast: 'uuid')]
        private Uuid $profileUuid,
        #[Column(type: 'integer')]
        private int $order,
        #[Embedded(target: Cost::class)]
        private Cost $cost,
        #[Embedded(target: Diff::class)]
        private Diff $diff,
        #[Embedded(target: Percents::class)]
        private Percents $percents,
        #[Column(type: 'text')]
        private string $callee,
        #[Column(type: 'text', nullable: true, default: null)]
        private ?string $caller = null,
        #[Column(type: 'string(36)', name: 'parent_uuid', nullable: true, default: null, typecast: 'uuid')]
        private ?Uuid $parentUuid = null,
    ) {}

    public function getUuid(): Uuid
    {
        return $this->uuid;
    }

    public function getProfileUuid(): Uuid
    {
        return $this->profileUuid;
    }

    public function getCost(): Cost
    {
        return $this->cost;
    }

    public function getDiff(): Diff
    {
        return $this->diff;
    }

    public function getPercents(): Percents
    {
        return $this->percents;
    }

    public function getCallee(): string
    {
        return $this->callee;
    }

    public function getCaller(): ?string
    {
        return $this->caller;
    }

    public function getParentUuid(): ?Uuid
    {
        return $this->parentUuid;
    }

    public function getOrder(): int
    {
        return $this->order;
    }
}
