<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\HttpDumps;

use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class HttpDumpsActionTest extends ControllerTestCase
{
    public function testHttpDumpsPost(): void
    {
        $this->http
            ->postJson(
                uri: '/',
                data: ['foo' => 'bar'],
                headers: ['X-Buggregator-Event' => 'http-dump'],
                cookies: ['foo' => 'bar'],
            )
            ->assertOk();

        $this->broadcastig->assertPushed('events', function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('http-dump', $data['data']['type']);
            $this->assertSame('POST', $data['data']['payload']['request']['method']);
            $this->assertSame('', $data['data']['payload']['request']['uri']);
            $this->assertSame(['http-dump'], $data['data']['payload']['request']['headers']['X-Buggregator-Event']);
            $this->assertSame('{"foo":"bar"}', $data['data']['payload']['request']['body']);
            $this->assertSame(['foo' => 'bar'], $data['data']['payload']['request']['cookies']);
            $this->assertSame([], $data['data']['payload']['request']['files']);
            $this->assertSame(['foo' => 'bar'], $data['data']['payload']['request']['post']);
            $this->assertSame([], $data['data']['payload']['request']['query']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);
            $this->assertNotEmpty($data['data']['payload']['received_at']);

            return true;
        });
    }

    public function testHttpDumpsGet(): void
    {
        $this->http
            ->getJson(
                uri: '/?bar=foo',
                query: ['foo' => 'bar'],
                headers: ['X-Buggregator-Event' => 'http-dump'],
                cookies: ['foo' => 'bar'],
            )
            ->assertOk();

        $this->broadcastig->assertPushed('events', function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('http-dump', $data['data']['type']);
            $this->assertSame('GET', $data['data']['payload']['request']['method']);
            $this->assertSame('', $data['data']['payload']['request']['uri']);
            $this->assertSame(['http-dump'], $data['data']['payload']['request']['headers']['X-Buggregator-Event']);
            $this->assertSame('{"foo":"bar"}', $data['data']['payload']['request']['body']);
            $this->assertSame(['foo' => 'bar'], $data['data']['payload']['request']['cookies']);
            $this->assertSame([], $data['data']['payload']['request']['files']);
            $this->assertSame(['foo' => 'bar'], $data['data']['payload']['request']['post']);
            $this->assertSame(['bar' => 'foo'], $data['data']['payload']['request']['query']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);
            $this->assertNotEmpty($data['data']['payload']['received_at']);

            return true;
        });
    }

    public function testHttpDumpsDelete(): void
    {
        $this->http
            ->deleteJson(
                uri: '/',
                data: ['foo' => 'bar'],
                headers: ['X-Buggregator-Event' => 'http-dump'],
                cookies: ['foo' => 'bar'],
            )
            ->assertOk();

        $this->broadcastig->assertPushed('events', function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('http-dump', $data['data']['type']);
            $this->assertSame('DELETE', $data['data']['payload']['request']['method']);
            $this->assertSame('', $data['data']['payload']['request']['uri']);
            $this->assertSame(['http-dump'], $data['data']['payload']['request']['headers']['X-Buggregator-Event']);
            $this->assertSame('{"foo":"bar"}', $data['data']['payload']['request']['body']);
            $this->assertSame(['foo' => 'bar'], $data['data']['payload']['request']['cookies']);
            $this->assertSame([], $data['data']['payload']['request']['files']);
            $this->assertSame(['foo' => 'bar'], $data['data']['payload']['request']['post']);
            $this->assertSame([], $data['data']['payload']['request']['query']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);
            $this->assertNotEmpty($data['data']['payload']['received_at']);

            return true;
        });
    }
}
