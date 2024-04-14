<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Ray;

use Nyholm\Psr7\Stream;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class RayActionTest extends ControllerTestCase
{
    public const PAYLOAD = <<<'JSON'
{"uuid":"11325003-b9cf-4c06-83d0-8a18fe368ac4","payloads":[{"type":"log","content":{"values":["foo"],"meta":[{"clipboard_data":"foo"}]},"origin":{"file":"\/root\/repos\/buggreagtor\/spiral-app\/tests\/Feature\/Interfaces\/Http\/Ray\/RayActionTest.php","line_number":13,"hostname":"ButschsterLpp"}}],"meta":{"php_version":"8.2.5","php_version_id":80205,"project_name":"","ray_package_version":"1.40.1.0"}}
JSON;

    public function testSendDump(): void
    {
        $this->http->postJson(
            uri: '/',
            data: Stream::create(self::PAYLOAD),
            headers: [
                'X-Buggregator-Event' => 'ray',
            ],
        )->assertOk();

        $this->assertEventSent();
    }

    public function testSendDumpViaHttpAuth(): void
    {
        $this->http->postJson(
            uri: 'http://ray@localhost',
            data: Stream::create(self::PAYLOAD),
        )->assertOk();

        $this->assertEventSent();
    }

    public function testSendDumpViaUserAgent(): void
    {
        $this->http
            ->withServerVariables([
                'HTTP_USER_AGENT' => 'ray 1.0.0',
            ])
            ->postJson(
            uri: '/',
            data: Stream::create(self::PAYLOAD),
        )->assertOk();

        $this->assertEventSent();
    }

    public function testSendDumpWithMerge(): void
    {
        $payload = self::PAYLOAD;
        $color = <<<'JSON'
{"uuid":"11325003-b9cf-4c06-83d0-8a18fe368ac4","payloads":[{"type":"color","content":{"color":"red"},"origin":{"file":"\/root\/repos\/buggreagtor\/spiral-app\/tests\/Feature\/Interfaces\/Http\/Ray\/RayActionTest.php","line_number":47,"hostname":"ButschsterLpp"}}],"meta":{"php_version":"8.2.5","php_version_id":80205,"project_name":"","ray_package_version":"1.40.1.0"}}
JSON;

        $this->http->postJson(
            uri: '/',
            data: Stream::create($payload),
            headers: ['X-Buggregator-Event' => 'ray',],
        )->assertOk();
        $this->broadcastig->reset();
        $this->http->postJson(
            uri: '/',
            data: Stream::create($color),
            headers: ['X-Buggregator-Event' => 'ray',],
        )->assertOk();

        $this->broadcastig->assertPushed('events', function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('ray', $data['data']['type']);

            $this->assertSame('11325003-b9cf-4c06-83d0-8a18fe368ac4', $data['data']['payload']['uuid']);
            $this->assertSame('8.2.5', $data['data']['payload']['meta']['php_version']);
            $this->assertSame('1.40.1.0', $data['data']['payload']['meta']['ray_package_version']);


            $this->assertSame('log', $data['data']['payload']['payloads'][0]['type']);
            $this->assertSame(['foo'], $data['data']['payload']['payloads'][0]['content']['values']);


            $this->assertSame('color', $data['data']['payload']['payloads'][1]['type']);
            $this->assertSame(['color' => 'red'], $data['data']['payload']['payloads'][1]['content']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }

    public function assertEventSent(): void
    {
        $this->broadcastig->assertPushed('events', function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('ray', $data['data']['type']);

            $this->assertSame('11325003-b9cf-4c06-83d0-8a18fe368ac4', $data['data']['payload']['uuid']);
            $this->assertSame('8.2.5', $data['data']['payload']['meta']['php_version']);
            $this->assertSame('1.40.1.0', $data['data']['payload']['meta']['ray_package_version']);


            $this->assertSame('log', $data['data']['payload']['payloads'][0]['type']);
            $this->assertSame(['foo'], $data['data']['payload']['payloads'][0]['content']['values']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }
}
