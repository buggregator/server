<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use Cycle\Annotated\Annotation\Relation\Embedded;
use Cycle\Annotated\Annotation\Relation\HasMany;
use Doctrine\Common\Collections\ArrayCollection;
use Modules\Profiler\Domain\Profile\Peaks;

// TODO: add repository
#[Entity(
    role: 'profile',
    table: 'profiles',
)]
class Profile
{
    #[HasMany(
        target: Edge::class,
        innerKey: 'uuid',
        outerKey: 'profile_uuid',
        orderBy: ['order' => 'ASC'],
        fkOnDelete: 'CASCADE'
    )]
    public ArrayCollection $edges;

    /** @internal */
    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: 'uuid')]
        private Uuid $uuid,
        #[Column(type: 'string')]
        private string $name,
        #[Embedded(target: Peaks::class)]
        private Peaks $peaks,
    ) {
        $this->edges = new ArrayCollection();
    }

    public function getUuid(): Uuid
    {
        return $this->uuid;
    }

    public function getName(): string
    {
        return $this->name;
    }

    public function getPeaks(): Peaks
    {
        return $this->peaks;
    }
}
