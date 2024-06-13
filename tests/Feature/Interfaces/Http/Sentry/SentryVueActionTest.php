<?php

declare(strict_types=1);

namespace Interfaces\Http\Sentry;

use App\Application\Broadcasting\Channel\EventsChannel;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ValueObject\Key;
use Nyholm\Psr7\Stream;
use Tests\App\Http\ResponseAssertions;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class SentryVueActionTest extends ControllerTestCase
{
    protected const JSON = <<<'BODY'
{"event_id":"9ac5bb343cfe45c19a7a59d45fb6ec89","sent_at":"2024-06-13T19:54:00.923Z","sdk":{"name":"sentry.javascript.vue","version":"8.9.2"},"trace":{"environment":"production","public_key":"sentry","trace_id":"8d53b11dcc1d47d2a02d2cbf0fe8c5bd","replay_id":"ad0f6efd15354b1b8886bed21fc64389","sample_rate":"1","transaction":"home","sampled":"true"}}
{"type":"event"}
{"exception":{"values":[{"type":"Error","value":"This is an error message","stacktrace":{"frames":[{"filename":"http://localhost:5173/src/main.js?t=1718307811541","function":"?","in_app":true,"lineno":38,"colno":5},{"filename":"http://localhost:5173/node_modules/.vite/deps/chunk-U6BEPC57.js?v=241240f6","function":"app.mount","in_app":true,"lineno":11123,"colno":19},{"filename":"http://localhost:5173/node_modules/.vite/deps/chunk-U6BEPC57.js?v=241240f6","function":"mount","in_app":true,"lineno":5379,"colno":13},{"filename":"http://localhost:5173/node_modules/.vite/deps/chunk-U6BEPC57.js?v=241240f6","function":"render2","in_app":true,"lineno":8118,"colno":7},{"filename":"http://localhost:5173/node_modules/.vite/deps/chunk-U6BEPC57.js?v=241240f6","function":"patch","in_app":true,"lineno":6795,"colno":11},{"filename":"http://localhost:5173/node_modules/.vite/deps/chunk-U6BEPC57.js?v=241240f6","function":"processComponent","in_app":true,"lineno":7329,"colno":9},{"filename":"http://localhost:5173/node_modules/.vite/deps/chunk-U6BEPC57.js?v=241240f6","function":"mountComponent","in_app":true,"lineno":7363,"colno":7},{"filename":"http://localhost:5173/node_modules/.vite/deps/chunk-U6BEPC57.js?v=241240f6","function":"setupComponent","in_app":true,"lineno":9034,"colno":36},{"filename":"http://localhost:5173/node_modules/.vite/deps/chunk-U6BEPC57.js?v=241240f6","function":"setupStatefulComponent","in_app":true,"lineno":9073,"colno":25},{"filename":"http://localhost:5173/node_modules/.vite/deps/chunk-U6BEPC57.js?v=241240f6","function":"callWithErrorHandling","in_app":true,"lineno":1659,"colno":19},{"filename":"http://localhost:5173/src/App.vue?t=1718307811541","function":"setup","in_app":true,"lineno":10,"colno":7}]},"mechanism":{"type":"generic","handled":false}}]},"level":"error","event_id":"9ac5bb343cfe45c19a7a59d45fb6ec89","platform":"javascript","request":{"url":"http://localhost:5173/","headers":{"User-Agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"}},"timestamp":1718308440.923,"environment":"production","sdk":{"integrations":["InboundFilters","FunctionToString","BrowserApiErrors","Breadcrumbs","GlobalHandlers","LinkedErrors","Dedupe","HttpContext","Vue","BrowserTracing","Replay"],"name":"sentry.javascript.vue","version":"8.9.2","packages":[{"name":"npm:@sentry/vue","version":"8.9.2"}]},"contexts":{"trace":{"span_id":"933583be4373d115","trace_id":"8d53b11dcc1d47d2a02d2cbf0fe8c5bd"},"vue":{"componentName":"<App>","lifecycleHook":"setup function","trace":"\n\n(found in <App> at /root/repos/apps/vue-project/src/App.vue)","propsData":{}}},"transaction":"home","breadcrumbs":[{"timestamp":1718308440.886,"category":"console","data":{"arguments":["[Vue warn]: Invalid vnode type when creating vnode: undefined.","\n"," at <App>"],"logger":"console"},"level":"warning","message":"[Vue warn]: Invalid vnode type when creating vnode: undefined. \n  at <App>"},{"timestamp":1718308440.886,"category":"console","data":{"arguments":["[Vue warn]: Invalid vnode type when creating vnode: undefined.","\n"," at <App>"],"logger":"console"},"level":"warning","message":"[Vue warn]: Invalid vnode type when creating vnode: undefined. \n  at <App>"},{"timestamp":1718308440.887,"category":"console","data":{"arguments":["[Vue warn]: Invalid vnode type when creating vnode: undefined.","\n"," at <App>"],"logger":"console"},"level":"warning","message":"[Vue warn]: Invalid vnode type when creating vnode: undefined. \n  at <App>"},{"timestamp":1718308440.887,"category":"console","data":{"arguments":["[Vue warn]: Invalid vnode type when creating vnode: undefined.","\n"," at <App>"],"logger":"console"},"level":"warning","message":"[Vue warn]: Invalid vnode type when creating vnode: undefined. \n  at <App>"},{"timestamp":1718308440.888,"category":"navigation","data":{"from":"/","to":"/"}}],"tags":{"replayId":"ad0f6efd15354b1b8886bed21fc64389"}}
BODY;

    private Project $project;

    protected function setUp(): void
    {
        parent::setUp();

        $this->project = $this->createProject('default');
    }

    public function testSend(): void
    {
        $this->makeRequest(project: $this->project->getKey())->assertOk();

        $this->broadcastig->assertPushed(new EventsChannel($this->project->getKey()), function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('sentry', $data['data']['type']);
            $this->assertSame('default', $data['data']['project']);

            $this->assertNull($data['data']['payload']['server_name']);
            $this->assertSame('production', $data['data']['payload']['environment']);
            $this->assertSame('error', $data['data']['payload']['level']);
            $this->assertSame('javascript', $data['data']['payload']['platform']);
            $this->assertSame('9ac5bb343cfe45c19a7a59d45fb6ec89', $data['data']['payload']['event_id']);


            $this->assertSame('This is an error message', $data['data']['payload']['exception']['values'][0]['value']);

            $this->assertCount(3, $data['data']['payload']['exception']['values'][0]['stacktrace']['frames']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }

    private function makeRequest(string $secret = 'secret', string|Key $project = 'default'): ResponseAssertions
    {
        return $this->http
            ->postJson(
                uri: '/api/' . $project . '/envelope/',
                data: Stream::create(self::JSON),
                headers: [
                    'X-Buggregator-Event' => 'sentry',
                    'Content-Type' => 'application/x-sentry-envelope',
                    'X-Sentry-Auth' => 'Sentry sentry_version=7, sentry_client=sentry.php/4.0.1, sentry_key=' . $secret,
                ],
            );
    }
}
