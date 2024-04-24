<?php

declare(strict_types=1);

namespace App\Application;

use Spiral\Core\Attribute\Singleton;

#[Singleton]
final readonly class AppVersion
{
    public function __construct(
        public string $version,
    ) {
    }
}
