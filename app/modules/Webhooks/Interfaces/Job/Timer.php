<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Job;

final readonly class Timer
{
    private \Closure $timer;

    public function __construct(?\Closure $timer = null)
    {
        $this->timer = $timer ?? function (int $seconds): void {
            \sleep($seconds);
        };
    }

    public function sleep(int $seconds): void
    {
        \call_user_func($this->timer, $seconds);
    }
}
