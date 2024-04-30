<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\TCP\VarDumper;

use App\Application\Broadcasting\Channel\EventsChannel;
use Modules\VarDumper\Application\Dump\DumpIdGeneratorInterface;
use Modules\VarDumper\Exception\InvalidPayloadException;
use Symfony\Component\VarDumper\Caster\ReflectionCaster;
use Symfony\Component\VarDumper\Cloner\VarCloner;
use Tests\Feature\Interfaces\TCP\TCPTestCase;

final class SymfonyV7Test extends TCPTestCase
{
    public function testSendDump(): void
    {
        $message = $this->buildPayload(var: 'foo');
        $this->handleVarDumperRequest($message);

        $this->broadcastig->assertPushed(new EventsChannel(), function (array $data) {
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

    public function testSendObjectDump(): void
    {
        $generator = $this->mockContainer(DumpIdGeneratorInterface::class);
        $generator->shouldReceive('generate')->andReturn('sf-dump-730421088');
        $object = (object)['type' => 'string', 'value' => 'foo'];
        $message = $this->buildPayload($object);
        $this->handleVarDumperRequest($message);
        $objectId = \spl_object_id($object);

        $this->broadcastig->assertPushed(new EventsChannel(), function (array $data) use ($objectId) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('var-dump', $data['data']['type']);
            $this->assertSame(null, $data['data']['project']);

            $this->assertSame([
                'type' => 'stdClass',
                'value' => \sprintf(
                    <<<HTML
<pre class=sf-dump id=sf-dump-730421088 data-indent-pad="  ">{<a class=sf-dump-ref>#%s</a><samp data-depth=1 class=sf-dump-expanded>
  +"<span class=sf-dump-public>type</span>": "<span class=sf-dump-str>string</span>"
  +"<span class=sf-dump-public>value</span>": "<span class=sf-dump-str>foo</span>"
</samp>}
</pre><script>Sfdump("sf-dump-730421088")</script>

HTML,
                    $objectId,
                ),
            ], $data['data']['payload']['payload']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }

    public function testSendDumpWithCodeHighlighting(): void
    {
        $message = $this->buildPayload(var: 'foo', context: ['language' => 'php']);
        $this->handleVarDumperRequest($message);

        $this->broadcastig->assertPushed(new EventsChannel(), function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('var-dump', $data['data']['type']);

            $this->assertSame([
                'type' => 'code',
                'value' => 'foo',
                'language' => 'php'
            ], $data['data']['payload']['payload']);

            return true;
        });
    }

    public function testSendDumpWithProject(): void
    {
        $this->createProject('default');
        $message = $this->buildPayload(project: 'default');
        $this->handleVarDumperRequest($message);

        $this->broadcastig->assertPushed(new EventsChannel('default'), function (array $data) {
            $this->assertSame('default', $data['data']['project']);
            return true;
        });
    }

    public function testSendDumpWithNonExistsProject(): void
    {
        $message = $this->buildPayload(project: 'default');
        $this->handleVarDumperRequest($message);

        $this->broadcastig->assertNotPushed(new EventsChannel('default'));
        $this->broadcastig->assertPushed(new EventsChannel(), function (array $data) {
            $this->assertSame(null, $data['data']['project']);
            return true;
        });
    }

    public function testSendInvalidDump(): void
    {
        $this->expectException(InvalidPayloadException::class);
        $this->expectExceptionMessage('Unable to decode the message.');

        $this->handleVarDumperRequest('invalid');
    }

    private function buildPayload(mixed $var = 'string', ?string $project = null, array $context = []): string
    {
        $cloner = new VarCloner();
        $cloner->addCasters(ReflectionCaster::UNSET_CLOSURE_FILE_INFO);
        $data = $cloner->cloneVar($var);

        if ($project !== null) {
            $context['project'] = $project;
        }

        return \base64_encode(\serialize([$data, $context])) . "\n";
    }
}
