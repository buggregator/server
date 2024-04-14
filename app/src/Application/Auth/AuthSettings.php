<?php

declare(strict_types=1);

namespace App\Application\Auth;

use Psr\Http\Message\UriInterface;

final readonly class AuthSettings
{
    public function __construct(
        public bool $enabled,
        public UriInterface $loginUrl,
    ) {}
}
