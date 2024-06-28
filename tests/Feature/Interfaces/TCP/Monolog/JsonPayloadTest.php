<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\TCP\Monolog;

use App\Application\Broadcasting\Channel\EventsChannel;
use Modules\Projects\Domain\ValueObject\Key;
use Tests\Feature\Interfaces\TCP\TCPTestCase;

final class JsonPayloadTest extends TCPTestCase
{
    public function testSendDump(): void
    {
        $message = \json_encode($payload = $this->buildMessage());
        $this->handleMonologRequest($message);

        $this->broadcastig->assertPushed(new EventsChannel(), function (array $data) use ($payload) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('monolog', $data['data']['type']);
            $this->assertSame(null, $data['data']['project']);

            $this->assertSame($payload, $data['data']['payload']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }

    public function testSendDumpWithProject(): void
    {
        $project = $this->createProject('default');
        $message = \json_encode($this->buildMessage($project->getKey()));

        $this->handleMonologRequest($message);

        $this->broadcastig->assertPushed(new EventsChannel('default'), function (array $data) {
            $this->assertSame('default', $data['data']['project']);
            return true;
        });
    }

    public function testSendDumpWithNonExistProject(): void
    {
        $message = \json_encode($this->buildMessage('default'));
        $this->handleMonologRequest($message);

        $this->broadcastig->assertNotPushed(new EventsChannel('default'));
        $this->broadcastig->assertPushed(new EventsChannel());
    }

    private function buildMessage(Key|string|null $project = null): array
    {
        $payload = [
            'message' => 'Some message',
            'context' => [],
            'level' => 400,
            'level_name' => 'ERROR',
            'channel' => 'socket',
            'datetime' => '2024-04-28T06:53:07.674031+00:00',
            'extra' => [],
        ];

        if ($project !== null) {
            $payload['context']['project'] = (string) $project;
        }

        return $payload;
    }
}
