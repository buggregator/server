<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Profiler;

use App\Application\Broadcasting\Channel\EventsChannel;
use Nyholm\Psr7\Stream;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class ProfilerActionTest extends ControllerTestCase
{
    public const PAYLOAD = <<<'JSON'
{"profile":{"Nyholm\\Psr7\\Response::getHeaderLine==\u003ENyholm\\Psr7\\Response::getHeader":{"ct":2,"wt":4,"cpu":4,"mu":648,"pmu":0},"App\\Middleware\\LocaleSelector::process==\u003ENyholm\\Psr7\\Response::getHeaderLine":{"ct":1,"wt":6,"cpu":6,"mu":1216,"pmu":0},"App\\Middleware\\LocaleSelector::process==\u003ENyholm\\Psr7\\Response::getBody":{"ct":1,"wt":1,"cpu":1,"mu":568,"pmu":0},"Nyholm\\Psr7\\Stream::getUri==\u003ENyholm\\Psr7\\Stream::getMetadata":{"ct":1,"wt":4,"cpu":5,"mu":1728,"pmu":0},"Nyholm\\Psr7\\Stream::getSize==\u003ENyholm\\Psr7\\Stream::getUri":{"ct":1,"wt":8,"cpu":8,"mu":1240,"pmu":0},"App\\Middleware\\LocaleSelector::process==\u003ENyholm\\Psr7\\Stream::getSize":{"ct":1,"wt":15,"cpu":16,"mu":3624,"pmu":0},"Spiral\\Telemetry\\Span::setStatus==\u003ESpiral\\Telemetry\\Span\\Status::__construct":{"ct":2,"wt":3,"cpu":3,"mu":584,"pmu":0},"App\\Middleware\\LocaleSelector::process==\u003ESpiral\\Telemetry\\Span::setStatus":{"ct":1,"wt":5,"cpu":6,"mu":1312,"pmu":0},"main()==\u003EApp\\Middleware\\LocaleSelector::process":{"ct":1,"wt":211752,"cpu":82707,"mu":2600352,"pmu":1837832},"main()==\u003ENyholm\\Psr7\\Response::getStatusCode":{"ct":2,"wt":1,"cpu":1,"mu":552,"pmu":0},"main()==\u003ESpiral\\Telemetry\\Span::setAttribute":{"ct":2,"wt":2,"cpu":3,"mu":928,"pmu":0},"main()==\u003ENyholm\\Psr7\\Response::getHeaderLine":{"ct":1,"wt":3,"cpu":4,"mu":552,"pmu":0},"main()==\u003ENyholm\\Psr7\\Response::getBody":{"ct":1,"wt":1,"cpu":1,"mu":536,"pmu":0},"main()==\u003ENyholm\\Psr7\\Stream::getSize":{"ct":1,"wt":0,"cpu":1,"mu":536,"pmu":0},"main()==\u003ESpiral\\Telemetry\\Span::setStatus":{"ct":1,"wt":2,"cpu":3,"mu":632,"pmu":0},"main()==\u003ESpiral\\Bootloader\\DebugBootloader::state":{"ct":1,"wt":96,"cpu":97,"mu":10200,"pmu":0},"main()==\u003ESpiral\\Debug\\State::getTags":{"ct":1,"wt":0,"cpu":1,"mu":536,"pmu":0},"main()==\u003ENyholm\\Psr7\\ServerRequest::getAttribute":{"ct":1,"wt":2,"cpu":2,"mu":552,"pmu":0},"main()":{"ct":1,"wt":211999,"cpu":82952,"mu":2614696,"pmu":1837832}},"tags":{"php":"8.2.5","dispatcher":"Spiral\\RoadRunnerBridge\\Http\\Dispatcher","method":"GET","url":"http:\/\/127.0.0.1:8080\/","route":null,"uri":"http:\/\/127.0.0.1:8080\/"},"app_name":"My super app","hostname":"Localhost","date":1714289301}
JSON;

    public function testSendData(): void
    {
        $this->http
            ->post(
                uri: '/api/profiler/store',
                data: Stream::create(self::PAYLOAD),
                headers: [
                    'X-Buggregator-Event' => 'profiler',
                ],
            )->assertOk();

        $this->assertEvent('default');
    }

    public function testSendDataWithProject(): void
    {
        $project = 'foo';
        $this->createProject($project);

        $this->http
            ->post(
                uri: '/api/profiler/store',
                data: Stream::create(self::PAYLOAD),
                headers: [
                    'X-Buggregator-Event' => 'profiler',
                    'X-Buggregator-Project' => $project,
                ],
            )->assertOk();

        $this->assertEvent($project);
    }

    public function testSendInvalidPayload(): void
    {
        $this->http
            ->post(
                uri: '/api/profiler/store',
                data: Stream::create(''),
                headers: [
                    'X-Buggregator-Event' => 'profiler',
                    'X-Buggregator-Project' => 'default',
                ],
            )->assertStatus(422);
    }

    public function assertEvent(?string $project = null): void
    {
        $this->broadcastig->assertPushed((string)new EventsChannel($project), function (array $data) use ($project) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('profiler', $data['data']['type']);
            $this->assertSame($project, $data['data']['project']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }
}
