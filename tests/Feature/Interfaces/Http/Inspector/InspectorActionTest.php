<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Inspector;

use Nyholm\Psr7\Stream;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class InspectorActionTest extends ControllerTestCase
{
    public function testSendData(): void
    {
        $this->http
            ->post(
                uri: '/',
                data: Stream::create(
                    <<<'BODY'
W3sibW9kZWwiOiJ0cmFuc2FjdGlvbiIsIm5hbWUiOiJcL2ZvbyIsInR5cGUiOiJwcm9jZXNzIiwiaGFzaCI6Ijk3OWZmYmNlY2ZjZDNhNzJjMWM0ZDUzNmFhMWZlODViM2U5OTZkZjFkNzA5Mzc1NWI5YjRhMWRlZDhlMzNiNWMiLCJob3N0Ijp7Imhvc3RuYW1lIjoiQnV0c2Noc3RlckxwcCIsImlwIjoiMTI3LjAuMS4xIiwib3MiOiJMaW51eCJ9LCJ0aW1lc3RhbXAiOjE3MDE0NjQwMzkuNjUwNjIyLCJtZW1vcnlfcGVhayI6MTUuNTMsImR1cmF0aW9uIjowLjIyfSx7Im1vZGVsIjoic2VnbWVudCIsInR5cGUiOiJteS1wcm9jZXNzIiwiaG9zdCI6eyJob3N0bmFtZSI6IkJ1dHNjaHN0ZXJMcHAiLCJpcCI6IjEyNy4wLjEuMSIsIm9zIjoiTGludXgifSwidHJhbnNhY3Rpb24iOnsibmFtZSI6IlwvZm9vIiwiaGFzaCI6Ijk3OWZmYmNlY2ZjZDNhNzJjMWM0ZDUzNmFhMWZlODViM2U5OTZkZjFkNzA5Mzc1NWI5YjRhMWRlZDhlMzNiNWMiLCJ0aW1lc3RhbXAiOjE3MDE0NjQwMzkuNjUwNjIyfSwic3RhcnQiOjAuMiwidGltZXN0YW1wIjoxNzAxNDY0MDM5LjY1MDgyNiwiZHVyYXRpb24iOjAuMDF9XQ==
BODY,
                ),
                headers: [
                    'X-Buggregator-Event' => 'inspector',
                    'X-Inspector-Key' => 'test',
                    'X-Inspector-Version' => '1.0.0',
                ],
            )->assertOk();

        $this->broadcastig->assertPushed('events', function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('inspector', $data['data']['type']);

            $this->assertSame('transaction', $data['data']['payload'][0]['model']);
            $this->assertSame('/foo', $data['data']['payload'][0]['name']);
            $this->assertSame('process', $data['data']['payload'][0]['type']);
            $this->assertSame(
                '979ffbcecfcd3a72c1c4d536aa1fe85b3e996df1d7093755b9b4a1ded8e33b5c',
                $data['data']['payload'][0]['hash'],
            );

            $this->assertSame('segment', $data['data']['payload'][1]['model']);
            $this->assertSame('my-process', $data['data']['payload'][1]['type']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }
}
