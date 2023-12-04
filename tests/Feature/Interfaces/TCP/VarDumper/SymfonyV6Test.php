<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\TCP\VarDumper;

use Modules\VarDumper\Interfaces\TCP\Service;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Tests\Feature\Interfaces\TCP\TCPTestCase;

final class SymfonyV6Test extends TCPTestCase
{
    public function testSendDump(): void
    {
        dump('foo');

        $payload = 'YToyOntpOjA7TzozOToiU3ltZm9ueVxDb21wb25lbnRcVmFyRHVtcGVyXENsb25lclxEYXRhIjo3OntzOjQ1OiIAU3ltZm9ueVxDb21wb25lbnRcVmFyRHVtcGVyXENsb25lclxEYXRhAGRhdGEiO2E6MTp7aTowO2E6MTp7aTowO3M6MzoiZm9vIjt9fXM6NDk6IgBTeW1mb255XENvbXBvbmVudFxWYXJEdW1wZXJcQ2xvbmVyXERhdGEAcG9zaXRpb24iO2k6MDtzOjQ0OiIAU3ltZm9ueVxDb21wb25lbnRcVmFyRHVtcGVyXENsb25lclxEYXRhAGtleSI7aTowO3M6NDk6IgBTeW1mb255XENvbXBvbmVudFxWYXJEdW1wZXJcQ2xvbmVyXERhdGEAbWF4RGVwdGgiO2k6MjA7czo1NzoiAFN5bWZvbnlcQ29tcG9uZW50XFZhckR1bXBlclxDbG9uZXJcRGF0YQBtYXhJdGVtc1BlckRlcHRoIjtpOi0xO3M6NTQ6IgBTeW1mb255XENvbXBvbmVudFxWYXJEdW1wZXJcQ2xvbmVyXERhdGEAdXNlUmVmSGFuZGxlcyI7aTotMTtzOjQ4OiIAU3ltZm9ueVxDb21wb25lbnRcVmFyRHVtcGVyXENsb25lclxEYXRhAGNvbnRleHQiO2E6MDp7fX1pOjE7YTozOntzOjk6InRpbWVzdGFtcCI7ZDoxNzAxNDk5NDM3LjUzODQ0NztzOjM6ImNsaSI7YToyOntzOjEyOiJjb21tYW5kX2xpbmUiO3M6MzIxOiIvcm9vdC9yZXBvcy9idWdncmVhZ3Rvci9zcGlyYWwtYXBwL3ZlbmRvci9waHB1bml0L3BocHVuaXQvcGhwdW5pdCAtLWNvbmZpZ3VyYXRpb24gL3Jvb3QvcmVwb3MvYnVnZ3JlYWd0b3Ivc3BpcmFsLWFwcC9waHB1bml0LnhtbCAtLWZpbHRlciAvKEludGVyZmFjZXNcXFRDUFxcVmFyRHVtcGVyXFxTeW1mb255VjZUZXN0Ojp0ZXN0U2VuZER1bXApKCAuKik/JC8gLS10ZXN0LXN1ZmZpeCBTeW1mb255VjZUZXN0LnBocCAvcm9vdC9yZXBvcy9idWdncmVhZ3Rvci9zcGlyYWwtYXBwL3Rlc3RzL0ZlYXR1cmUvSW50ZXJmYWNlcy9UQ1AvVmFyRHVtcGVyIC0tdGVhbWNpdHkiO3M6MTA6ImlkZW50aWZpZXIiO3M6ODoiZGVlMTBhZWUiO31zOjY6InNvdXJjZSI7YTo0OntzOjQ6Im5hbWUiO3M6MTc6IlN5bWZvbnlWNlRlc3QucGhwIjtzOjQ6ImZpbGUiO3M6OTE6Ii9yb290L3JlcG9zL2J1Z2dyZWFndG9yL3NwaXJhbC1hcHAvdGVzdHMvRmVhdHVyZS9JbnRlcmZhY2VzL1RDUC9WYXJEdW1wZXIvU3ltZm9ueVY2VGVzdC5waHAiO3M6NDoibGluZSI7aToxMztzOjEyOiJmaWxlX2V4Y2VycHQiO2I6MDt9fX0=';

        $service = $this->get(Service::class);

        $service->handle(
            new Request(
                remoteAddr: '127.0.0.1',
                event: TcpEvent::Data,
                body: $payload,
                connectionUuid: (string)$this->randomUuid(),
                server: 'local',
            ),
        );

        $this->broadcastig->assertPushed('events', function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('var-dump', $data['data']['type']);

            $this->assertSame([
                'type' => 'string',
                'value' => 'foo',
            ], $data['data']['payload']['payload']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);


            return true;
        });
    }
}
