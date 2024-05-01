<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Ray;

use Spatie\Ray\Ray;
use Tests\App\Http\HttpFaker;
use Tests\App\Ray\FakeClient;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

abstract class RayTestCase extends ControllerTestCase
{
    private FakeClient $client;

    protected function setUp(): void
    {
        parent::setUp();

        $this->client = new FakeClient(new HttpFaker($this->fakeHttp()));
    }

    protected function buildRay(): Ray
    {
        $this->client->clearRequests();

        return Ray::create(client: $this->client);
    }

    protected function dump(mixed ...$args): Ray
    {
        return $this->buildRay()->send(...$args);
    }

    protected function getSentRequests(): array
    {
        return $this->client->getSentRequests();
    }
}
