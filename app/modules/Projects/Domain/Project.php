<?php

declare(strict_types=1);

namespace Modules\Projects\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;

#[Entity(
    repository: ProjectRepositoryInterface::class
)]
class Project
{
    /**  @internal */
    public function __construct(
        #[Column(type: 'string(36)', primary: true)]
        private string $key,

        #[Column(type: 'string')]
        private string $name,
    ) {
    }

    public function getKey(): string
    {
        return $this->key;
    }

    public function getName(): string
    {
        return $this->name;
    }
}
