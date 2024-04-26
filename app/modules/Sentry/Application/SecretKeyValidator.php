<?php

declare(strict_types=1);

namespace Modules\Sentry\Application;

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

        if (isset($request->getQueryParams()['sentry_key'])) {
            return $this->secret === $request->getQueryParams()['sentry_key'];
        }

        if (!$request->hasHeader('X-Sentry-Auth')) {
            return false;
        }

        $key = null;
        $parts = \explode(',', $request->getHeaderLine('X-Sentry-Auth'));
        foreach ($parts as $part) {
            if (\str_starts_with($part, 'sentry_key=')) {
                $key = \substr($part, 11);
                break;
            }
        }

        return $this->secret === $key;
    }
}
