<?php

declare(strict_types=1);

namespace App\Application\Client;

use Spiral\Core\Attribute\Singleton;

#[Singleton]
final readonly class ClientSettings
{
    public function __construct(
        public string $supportedEvents,
    ) {}
}
