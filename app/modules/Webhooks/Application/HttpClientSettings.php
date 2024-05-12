<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use Spiral\Boot\EnvironmentInterface;

final readonly class HttpClientSettings
{
    public function __construct(
        private EnvironmentInterface $env,
    ) {}

    public function getUserAgent(): string
    {
        return $this->env->get('WEBHOOK_USER_AGENT', 'Buggregator\Webhooks');
    }

    public function getTimeout(): int
    {
        return (int) $this->env->get('WEBHOOK_TIMEOUT', 10);
    }

    public function getContentType(): string
    {
        return $this->env->get('WEBHOOK_CONTENT_TYPE', 'application/json');
    }
}
