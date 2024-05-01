<?php

declare(strict_types=1);

namespace Tests\App\Ray;

use Nyholm\Psr7\Stream;
use Spatie\Ray\Client;
use Spatie\Ray\Request;
use Tests\App\Http\HttpFaker;

final class FakeClient extends Client
{
    /** @var Request[] */
    private array $sentRequests = [];

    public function __construct(
        private readonly HttpFaker $http,
    ) {}

    public function performAvailabilityCheck(): bool
    {
        return true;
    }

    public function serverIsAvailable(): bool
    {
        return true;
    }

    public function lockExists(string $lockName): bool
    {
        return false;
    }

    public function send(Request $request): void
    {
        $this->sentRequests[] = $request;

        $this->http->postJson(
            uri: '/',
            data: Stream::create($request->toJson()),
            headers: ['X-Buggregator-Event' => 'ray'],
        )->assertOk();
    }

    public function __destruct()
    {
        // do nothing
    }

    public function getSentRequests(): array
    {
        return $this->sentRequests;
    }

    public function clearRequests(): void
    {
        $this->sentRequests = [];
    }
}
