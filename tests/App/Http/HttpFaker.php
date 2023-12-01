<?php

declare(strict_types=1);

namespace Tests\App\Http;

use App\Application\Domain\ValueObjects\Uuid;
use Carbon\Carbon;
use Ramsey\Uuid\UuidInterface;
use Spiral\Testing\Http\FakeHttp;
use Spiral\Testing\Http\TestResponse;
use Tests\TestCase;

/**
 * @mixin FakeHttp
 */
final class HttpFaker
{
    private Carbon $date;
    private bool $dumpResponse = false;

    public function __construct(
        private FakeHttp $http,
        private TestCase $tests,
    ) {
        $this->date = Carbon::create(2021, 1, 1, 0, 0, 0);
    }

    public function showEvent(Uuid $uuid): ResponseAssertions
    {
        return $this->makeResponse(
            $this->http->getJson(uri: '/api/event/' . $uuid),
        );
    }

    public function deleteEvent(Uuid $uuid): ResponseAssertions
    {
        return $this->makeResponse(
            $this->http->deleteJson(uri: '/api/event/' . $uuid),
        );
    }

    public function clearEvents(?string $type = null): ResponseAssertions
    {
        return $this->makeResponse(
            $this->http->deleteJson(
                uri: '/api/events/',
                data: $type ? ['type' => $type] : [],
            ),
        );
    }

    public function __call(string $name, array $arguments): ResponseAssertions
    {
        if (!method_exists($this->http, $name)) {
            throw new \Exception("Method $name does not exist");
        }

        return $this->makeResponse(
            $this->http->$name(...$arguments),
        );
    }

    private function makeResponse(TestResponse $response): ResponseAssertions
    {
        if ($this->dumpResponse) {
            $body = (string)$response;

            try {
                $body = \json_decode($body, true, 512, JSON_THROW_ON_ERROR);
            } catch (\Throwable) {
            }
        }

        return new ResponseAssertions($response);
    }
}
