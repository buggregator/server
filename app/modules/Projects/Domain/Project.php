<?php

declare(strict_types=1);

namespace Modules\Projects\Domain;

use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use Modules\Projects\Domain\ValueObject\Key;

#[Entity(
    repository: ProjectRepositoryInterface::class
)]
class Project
{
    /**  @internal */
    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: Key::class)]
        private Key $key,

        #[Column(type: 'string')]
        private string $name,
    ) {
    }

    public function getKey(): Key
    {
        return $this->key;
    }

    public function getName(): string
    {
        return $this->name;
    }
}
