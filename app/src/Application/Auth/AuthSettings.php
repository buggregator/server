<?php

declare(strict_types=1);

namespace App\Application\Auth;

use Psr\Http\Message\UriInterface;

final class AuthSettings
{
    public function __construct(
        public readonly bool $enabled,
        public readonly UriInterface $loginUrl,
    ) {
    }
}
