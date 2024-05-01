<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\VarDumper;

use Modules\VarDumper\Application\Dump\DumpIdGeneratorInterface;
use Modules\VarDumper\Application\Dump\HtmlDumper;
use Symfony\Component\VarDumper\Caster\ReflectionCaster;
use Symfony\Component\VarDumper\Cloner\VarCloner;
use Tests\TestCase;

final class VarDumperTest extends TestCase
{
    public function testMaxDepth(): void
    {
        $array = (object) [
            'foo' => [
                'bar' => [
                    'baz' => [
                        'qux' => [
                            'quux' => 'quuz',
                        ],
                    ],
                ],
            ],
        ];

        $cloner = new VarCloner();
        $cloner->addCasters(ReflectionCaster::UNSET_CLOSURE_FILE_INFO);
        $data = $cloner->cloneVar($array);
        $objectId = \spl_object_id($array);

        $idGenerator = $this->mockContainer(DumpIdGeneratorInterface::class);
        $idGenerator->shouldReceive('generate')->andReturn('sf-dump-15682422');

        $dumper = new HtmlDumper(
            generator: $idGenerator,
            maxDepth: 2,
        );

        $this->assertSame(
            <<<HTML
<pre class=sf-dump id=sf-dump-15682422 data-indent-pad="  ">{<a class=sf-dump-ref>#{$objectId}</a><samp data-depth=1 class=sf-dump-expanded>
  +"<span class=sf-dump-public>foo</span>": <span class=sf-dump-note>array:1</span> [<samp data-depth=2 class=sf-dump-compact>
    "<span class=sf-dump-key>bar</span>" => <span class=sf-dump-note>array:1</span> [ &#8230;1]
  </samp>]
</samp>}
</pre><script>Sfdump("sf-dump-15682422")</script>

HTML,
            $dumper->dump($data, true),
        );
    }
}
