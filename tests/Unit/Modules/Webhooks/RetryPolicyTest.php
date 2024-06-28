<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Webhooks;

use Modules\Webhooks\Interfaces\Job\RetryPolicy;
use Modules\Webhooks\Interfaces\Job\Timer;
use Tests\TestCase;

final class RetryPolicyTest extends TestCase
{
    public function testCanTry(): void
    {
        $i = 0;
        $policy = new RetryPolicy(
            timer: new Timer(function (int $seconds) use (&$i): void {
                $expected = match ($i) {
                    0 => 5,
                    1 => 10,
                    2 => 20,
                    default => 0,
                };

                $this->assertSame($expected, $seconds);
                ++$i;
            }),
            maxRetries: 3,
            delay: 5,
            retryMultiplier: 2,
        );

        $this->assertTrue($policy->canRetry());
        $policy->nextRetry();
        $this->assertTrue($policy->canRetry());
        $policy->nextRetry();
        $this->assertTrue($policy->canRetry());
        $policy->nextRetry();
        $this->assertFalse($policy->canRetry());
    }

    public function testCanTryWithZeroRetries(): void
    {
        $policy = new RetryPolicy(
            timer: new Timer(function (int $seconds): never {
                $this->fail('No retries should be made');
            }),
            maxRetries: 0,
            delay: 5,
            retryMultiplier: 2,
        );

        $this->assertFalse($policy->canRetry());
    }

    public function testCanTryWithOneRetry(): void
    {
        $policy = new RetryPolicy(
            timer: new Timer(static function (int $seconds) : void {
            }),
            maxRetries: 1,
            delay: 5,
            retryMultiplier: 2,
        );

        $this->assertTrue($policy->canRetry());
        $policy->nextRetry();
        $this->assertFalse($policy->canRetry());
    }
}
