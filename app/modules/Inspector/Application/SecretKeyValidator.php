<?php

declare(strict_types=1);

namespace Modules\Inspector\Application;

use Psr\Http\Message\ServerRequestInterface;
use Spiral\Core\Attribute\Singleton;

#[Singleton]
final readonly class SecretKeyValidator
{
    public function __construct(
        private ?string $secret = null,
    ) {
    }

    public function validateRequest(ServerRequestInterface $request): bool
    {
        if ($this->secret === null) {
            return true;
        }

        if (!$request->hasHeader('X-Inspector-Key')) {
            return false;
        }

        return $this->secret === $request->getHeaderLine('X-Inspector-Key');
    }
}
