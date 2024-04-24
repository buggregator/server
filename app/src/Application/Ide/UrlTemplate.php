<?php

declare(strict_types=1);

namespace App\Application\Ide;

use Spiral\Core\Attribute\Singleton;

#[Singleton]
final readonly class UrlTemplate
{
    public function __construct(
        public string $template,
    ) {
    }
}
