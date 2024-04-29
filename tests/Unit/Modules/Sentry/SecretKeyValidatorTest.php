<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Sentry;

use Modules\Sentry\Application\SecretKeyValidator;
use Nyholm\Psr7\ServerRequest;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Testing\Attribute\Env;
use Tests\TestCase;

final class SecretKeyValidatorTest extends TestCase
{
    private const AUTH_HEADER = 'Sentry sentry_version=7, sentry_client=raven-php/0.15.0, sentry_key=1234567890';

    #[Env('SENTRY_SECRET_KEY', '1234567890')]
    public function testValidateHeaderKey(): void
    {
        $this->assertTrue(
            $this->get(SecretKeyValidator::class)->validateRequest(
                $this->createServerRequest(
                    headers: ['X-Sentry-Auth' => self::AUTH_HEADER],
                ),
            ),
        );
    }

    #[Env('SENTRY_SECRET_KEY', '123')]
    public function testValidateWrongHeaderKey(): void
    {
        $this->assertFalse(
            $this->get(SecretKeyValidator::class)->validateRequest(
                $this->createServerRequest(
                    headers: ['X-Sentry-Auth' => self::AUTH_HEADER],
                ),
            ),
        );
    }

    #[Env('SENTRY_SECRET_KEY', null)]
    public function testValidateWithoutSpecifiedKey(): void
    {
        $this->assertTrue(
            $this->get(SecretKeyValidator::class)->validateRequest(
                $this->createServerRequest(
                    headers: ['X-Sentry-Auth' => self::AUTH_HEADER],
                ),
            ),
        );

        $this->assertTrue(
            $this->get(SecretKeyValidator::class)->validateRequest(
                $this->createServerRequest(
                    query: ['sentry_key' => '1234567890'],
                ),
            ),
        );
    }

    #[Env('SENTRY_SECRET_KEY', '1234567890')]
    public function testValidateQueryKey(): void
    {
        $this->assertTrue(
            $this->get(SecretKeyValidator::class)->validateRequest(
                $this->createServerRequest(
                    query: ['sentry_key' => '1234567890'],
                ),
            ),
        );
    }

    #[Env('SENTRY_SECRET_KEY', '123')]
    public function testValidateWrongQueryKey(): void
    {
        $this->assertFalse(
            $this->get(SecretKeyValidator::class)->validateRequest(
                $this->createServerRequest(
                    query: ['sentry_key' => '1234567890'],
                ),
            ),
        );
    }

    private function createServerRequest(array $headers = [], array $query = []): ServerRequestInterface
    {
        return (new ServerRequest('GET', 'http://localhost', $headers))
            ->withQueryParams($query);
    }
}
