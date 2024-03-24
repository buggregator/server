<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;

#[Entity(
    repository: ConnectionRepositoryInterface::class
)]
class Connection
{
    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: 'uuid')]
        public Uuid $uuid,
        #[Column(type: 'string')]
        public string $name,
        #[Column(type: 'string')]
        public string $host,
        #[Column(type: 'string')]
        public string $user = 'root',
        #[Column(type: 'integer')]
        public int $port = 22,
        #[Column(type: 'string', nullable: true, default: null)]
        public ?string $password = null,
        #[Column(type: 'string', nullable: true, default: null)]
        public ?string $privateKey = null,
    ) {
    }
}
