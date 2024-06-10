<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\TCP\VarDumper;

use App\Application\Broadcasting\Channel\EventsChannel;
use Modules\VarDumper\Application\Dump\DumpIdGeneratorInterface;
use Modules\VarDumper\Exception\InvalidPayloadException;
use PHPUnit\Framework\Attributes\DataProvider;
use Symfony\Component\VarDumper\Caster\ReflectionCaster;
use Symfony\Component\VarDumper\Cloner\VarCloner;
use Tests\Feature\Interfaces\TCP\TCPTestCase;

final class SymfonyV7Test extends TCPTestCase
{
    public static function variablesDataProvider(): iterable
    {
        yield 'string' => ['foo', 'string', 'foo'];
        yield 'true' => [true, 'boolean', '1'];
        yield 'false' => [false, 'boolean', '0'];
        yield 'int' => [1, 'integer', '1'];
        yield 'float' => [1.1, 'double', '1.1'];
        yield 'array' => [
            ['foo' => 'bar'],
            'array',
            <<<'HTML'
<pre class=sf-dump id=sf-dump-730421088 data-indent-pad="  "><span class=sf-dump-label>Some label</span> <span class=sf-dump-note>array:1</span> [<samp data-depth=1 class=sf-dump-expanded>
  "<span class=sf-dump-key>foo</span>" => "<span class=sf-dump-str>bar</span>"
</samp>]
</pre><script>Sfdump("sf-dump-730421088")</script>

HTML
            ,
        ];
        yield 'object' => [
            (object) ['type' => 'string', 'value' => 'foo'],
            'stdClass',
            <<<HTML
<pre class=sf-dump id=sf-dump-730421088 data-indent-pad="  "><span class=sf-dump-label>Some label</span> {<a class=sf-dump-ref>#%s</a><samp data-depth=1 class=sf-dump-expanded>
  +"<span class=sf-dump-public>type</span>": "<span class=sf-dump-str>string</span>"
  +"<span class=sf-dump-public>value</span>": "<span class=sf-dump-str>foo</span>"
</samp>}
</pre><script>Sfdump("sf-dump-730421088")</script>

HTML
            ,
        ];
    }

    #[DataProvider('variablesDataProvider')]
    public function testSendDump(mixed $value, string $type, mixed $expected): void
    {
        $generator = $this->mockContainer(DumpIdGeneratorInterface::class);
        $generator->shouldReceive('generate')->andReturn('sf-dump-730421088');

        $message = $this->buildPayload(var: $value);
        $this->handleVarDumperRequest($message);

        if (\is_object($value)) {
            $expected = \sprintf($expected, \spl_object_id($value));
        }

        $this->broadcastig->assertPushed(new EventsChannel(), function (array $data) use ($value, $type, $expected) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('var-dump', $data['data']['type']);

            $this->assertSame([
                'type' => $type,
                'value' => $expected,
                'label' => 'Some label',
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
                'label' => 'Some label',
                'language' => 'php',
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

        $context['label'] = 'Some label';

        return \base64_encode(\serialize([$data->withContext($context), []])) . "\n";
    }
}
