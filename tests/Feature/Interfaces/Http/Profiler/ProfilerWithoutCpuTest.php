<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Profiler;

use App\Application\Broadcasting\Channel\EventsChannel;
use Modules\Profiler\Application\MetricsHelper;
use Nyholm\Psr7\Stream;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class ProfilerWithoutCpuTest extends ControllerTestCase
{
    /**
     * Profile payload without CPU metrics (when XHPROF_FLAGS_CPU is not enabled)
     */
    public const PAYLOAD_WITHOUT_CPU = <<<'JSON'
                     {"profile":{"Nyholm\\Psr7\\Response::getHeaderLine==\u003ENyholm\\Psr7\\Response::getHeader":{"ct":2,"wt":4,"mu":648,"pmu":0},"App\\Middleware\\LocaleSelector::process==\u003ENyholm\\Psr7\\Response::getHeaderLine":{"ct":1,"wt":6,"mu":1216,"pmu":0},"App\\Middleware\\LocaleSelector::process==\u003ENyholm\\Psr7\\Response::getBody":{"ct":1,"wt":1,"mu":568,"pmu":0},"main()==\u003EApp\\Middleware\\LocaleSelector::process":{"ct":1,"wt":211752,"mu":2600352,"pmu":1837832},"main()==\u003ENyholm\\Psr7\\Response::getStatusCode":{"ct":2,"wt":1,"mu":552,"pmu":0},"main()":{"ct":1,"wt":211999,"mu":2614696,"pmu":1837832}},"tags":{"php":"8.2.5","dispatcher":"Test"},"app_name":"Test App","hostname":"Test","date":1714289301}
                     JSON;

    public function testSendDataWithoutCpuMetrics(): void
    {
        $this->http
            ->post(
                uri: '/api/profiler/store',
                data: Stream::create(self::PAYLOAD_WITHOUT_CPU),
                headers: [
                    'X-Buggregator-Event' => 'profiler',
                ],
            )->assertOk();

        $this->assertEvent('default');
    }

    public function testMetricsHelperHandlesMissingCpu(): void
    {
        $dataWithoutCpu = ['wt' => 100, 'mu' => 1024];
        $normalized = MetricsHelper::getAllMetrics($dataWithoutCpu);

        $this->assertSame(0, $normalized['cpu']);
        $this->assertSame(100, $normalized['wt']);
        $this->assertSame(1024, $normalized['mu']);
        $this->assertSame(0, $normalized['pmu']);
        $this->assertSame(0, $normalized['ct']);
    }

    public function testHasCpuMetricsDetection(): void
    {
        $this->assertFalse(MetricsHelper::hasCpuMetrics([]));
        $this->assertFalse(MetricsHelper::hasCpuMetrics(['cpu' => 0]));
        $this->assertTrue(MetricsHelper::hasCpuMetrics(['cpu' => 100]));
    }

    private function assertEvent(?string $project = null): void
    {
        $this->broadcastig->assertPushed((string) new EventsChannel($project), function (array $data) use ($project) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('profiler', $data['data']['type']);
            $this->assertSame($project, $data['data']['project']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }
}
