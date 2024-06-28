<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Job;

class RetryPolicy
{
    private int $currentRetry = 0;

    public function __construct(
        private readonly Timer $timer,
        private int $maxRetries = 3,
        private readonly int $delay = 5,
        private readonly float|int $retryMultiplier = 2,
    ) {}

    public function setMaxRetries(int $maxRetries): self
    {
        $this->maxRetries = $maxRetries;
        return $this;
    }

    public function canRetry(): bool
    {
        return $this->currentRetry < $this->maxRetries;
    }

    public function nextRetry(): void
    {
        if (!$this->canRetry()) {
            throw new \RuntimeException('No more retries allowed');
        }

        $this->timer->sleep($this->getDelay());
        ++$this->currentRetry;
    }

    private function getDelay(): int
    {
        return (int) ($this->delay * ($this->retryMultiplier ** $this->currentRetry));
    }
}
