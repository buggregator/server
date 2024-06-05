<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Sentry;

use PHPUnit\Framework\Attributes\DataProvider;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class JavascriptActionTest extends ControllerTestCase
{
    public static function projectNameDataProvider(): iterable
    {
        yield 'string' => ['abc'];
        yield 'int-string' => ['123'];
    }

    #[DataProvider('projectNameDataProvider')]
    public function testCallAction(string $name): void
    {
        $response = $this->http->get('/api/sentry/' . $name . '.js');

        $response->assertOk();

        $response->assertBodyContains(
            \sprintf(
                <<<'JS'
"Sentry",'%s','https://browser.sentry-cdn.com/7.69.0/bundle.tracing.replay.min.js',{"dsn":"http://sentry@127.0.0.1:8000/%s"
JS,
                $name,
                $name,
            ),
        );
    }
}
