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
    ) {}

    public function validateRequest(ServerRequestInterface $request): bool
    {
        if (empty($this->secret)) {
            return true;
        }

        if (isset($request->getQueryParams()['sentry_key'])) {
            return $this->secret === $request->getQueryParams()['sentry_key'];
        }

        if (!$request->hasHeader('X-Sentry-Auth')) {
            return false;
        }

        // Header: Sentry sentry_version=7, sentry_client=raven-php/0.15.0, sentry_key=1234567890
        $key = \preg_match(
            '/sentry_key=(\w+)/',
            $request->getHeaderLine('X-Sentry-Auth'),
            $matches,
        ) ? $matches[1] : null;

        return $this->secret === $key;
    }
}
