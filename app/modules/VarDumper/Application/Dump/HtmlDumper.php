<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application\Dump;

use Symfony\Component\VarDumper\Cloner\Cursor;
use Symfony\Component\VarDumper\Cloner\Data;
use Symfony\Component\VarDumper\Dumper\AbstractDumper;
use Symfony\Component\VarDumper\Dumper\CliDumper;

final class HtmlDumper extends CliDumper
{
    /** @var callable|resource|string|null */
    public static $defaultOutput = 'php://output';

    protected string $dumpPrefix = '<pre class=sf-dump id=%s data-indent-pad="%s">';
    protected string $dumpSuffix = '</pre><script>Sfdump(%s)</script>';
    protected string $dumpId = 'sf-dump';
    protected $colors = true;
    protected int $lastDepth = -1;

    private array $displayOptions = [
        'maxDepth' => 1,
        'maxStringLength' => 160,
        'fileLinkFormat' => null,
    ];
    private array $extraDisplayOptions = [];

    public function __construct(
        DumpIdGeneratorInterface $generator = new MtRandDumpIdGenerator(),
        private int $maxDepth = 20,
    ) {
        AbstractDumper::__construct(null, null, 0);
        $this->dumpId = $generator->generate();
        $this->displayOptions['fileLinkFormat'] = \ini_get('xdebug.file_link_format') ?: get_cfg_var(
            'xdebug.file_link_format',
        );
    }

    public function setStyles(array $styles): void
    {
        $this->styles = $styles + $this->styles;
    }

    /**
     * Configures display options.
     *
     * @param array $displayOptions A map of display options to customize the behavior
     */
    public function setDisplayOptions(array $displayOptions): void
    {
        $this->displayOptions = $displayOptions + $this->displayOptions;
    }

    public function dump(Data $data, $output = true, array $extraDisplayOptions = []): ?string
    {
        $this->extraDisplayOptions = $extraDisplayOptions;
        $result = parent::dump($data->withMaxDepth($this->maxDepth), $output);
        $this->dumpId = 'sf-dump-' . mt_rand();

        return $result;
    }

    public function dumpString(Cursor $cursor, string $str, bool $bin, int $cut): void
    {
        if ('' === $str && isset($cursor->attr['img-data'], $cursor->attr['content-type'])) {
            $this->dumpKey($cursor);
            $this->line .= $this->style('default', $cursor->attr['img-size'] ?? '', []);
            $this->line .= $cursor->depth >= $this->displayOptions['maxDepth'] ? ' <samp class=sf-dump-compact>' : ' <samp class=sf-dump-expanded>';
            $this->endValue($cursor);
            $this->line .= $this->indentPad;
            $this->line .= \sprintf(
                '<img src="data:%s;base64,%s" /></samp>',
                $cursor->attr['content-type'],
                \base64_encode($cursor->attr['img-data']),
            );
            $this->endValue($cursor);
        } else {
            parent::dumpString($cursor, $str, $bin, $cut);
        }
    }

    public function enterHash(Cursor $cursor, int $type, string|int|null $class, bool $hasChild): void
    {
        if (Cursor::HASH_OBJECT === $type) {
            $cursor->attr['depth'] = $cursor->depth;
        }

        parent::enterHash($cursor, $type, $class, false);

        if ($cursor->skipChildren || $cursor->depth >= $this->displayOptions['maxDepth']) {
            $cursor->skipChildren = false;
            $eol = ' class=sf-dump-compact>';
        } else {
            $this->expandNextHash = false;
            $eol = ' class=sf-dump-expanded>';
        }

        if ($hasChild) {
            $this->line .= '<samp data-depth=' . ($cursor->depth + 1);
            if ($cursor->refIndex) {
                $r = Cursor::HASH_OBJECT !== $type ? 1 - (Cursor::HASH_RESOURCE !== $type) : 2;
                $r .= $r && 0 < $cursor->softRefHandle ? $cursor->softRefHandle : $cursor->refIndex;

                $this->line .= sprintf(' id=%s-ref%s', $this->dumpId, $r);
            }
            $this->line .= $eol;
            $this->dumpLine($cursor->depth);
        }
    }

    public function leaveHash(Cursor $cursor, int $type, string|int|null $class, bool $hasChild, int $cut): void
    {
        $this->dumpEllipsis($cursor, $hasChild, $cut);
        if ($hasChild) {
            $this->line .= '</samp>';
        }
        parent::leaveHash($cursor, $type, $class, $hasChild, 0);
    }

    protected function style(string $style, string $value, array $attr = []): string
    {
        if ('' === $value && ('label' !== $style || !isset($attr['file']) && !isset($attr['href']))) {
            return '';
        }

        $v = esc($value);

        if ('ref' === $style) {
            if (empty($attr['count'])) {
                return sprintf('<a class=sf-dump-ref>%s</a>', $v);
            }
            $r = ('#' !== $v[0] ? 1 - ('@' !== $v[0]) : 2) . substr($value, 1);

            return sprintf(
                '<a class=sf-dump-ref href=#%s-ref%s title="%d occurrences">%s</a>',
                $this->dumpId,
                $r,
                1 + $attr['count'],
                $v,
            );
        }

        if ('const' === $style && isset($attr['value'])) {
            $style .= sprintf(
                ' title="%s"',
                esc(\is_scalar($attr['value']) ? $attr['value'] : json_encode($attr['value'])),
            );
        } elseif ('note' === $style && 0 < ($attr['depth'] ?? 0) && false !== $c = strrpos($value, '\\')) {
            $style .= ' title=""';
            $attr += [
                'ellipsis' => \strlen($value) - $c,
                'ellipsis-type' => 'note',
                'ellipsis-tail' => 1,
            ];
        }

        if (isset($attr['ellipsis'])) {
            $class = 'sf-dump-ellipsis';
            if (isset($attr['ellipsis-type'])) {
                $class = sprintf('"%s sf-dump-ellipsis-%s"', $class, $attr['ellipsis-type']);
            }
            $label = esc(substr($value, -$attr['ellipsis']));
            $style = str_replace(' title="', " title=\"$v\n", $style);
            $v = sprintf('<span class=%s>%s</span>', $class, substr($v, 0, -\strlen($label)));

            if (!empty($attr['ellipsis-tail'])) {
                $tail = \strlen(esc(substr($value, -$attr['ellipsis'], $attr['ellipsis-tail'])));
                $v .= sprintf('<span class=%s>%s</span>%s', $class, substr($label, 0, $tail), substr($label, $tail));
            } else {
                $v .= $label;
            }
        }

        $map = static::$controlCharsMap;
        $v = "<span class=sf-dump-{$style}>" . preg_replace_callback(static::$controlCharsRx, function ($c) use ($map) {
            $s = $b = '<span class="sf-dump-default';
            $c = $c[$i = 0];
            if ($ns = "\r" === $c[$i] || "\n" === $c[$i]) {
                $s .= ' sf-dump-ns';
            }
            $s .= '">';
            do {
                if (("\r" === $c[$i] || "\n" === $c[$i]) !== $ns) {
                    $s .= '</span>' . $b;
                    if ($ns = !$ns) {
                        $s .= ' sf-dump-ns';
                    }
                    $s .= '">';
                }

                $s .= $map[$c[$i]] ?? sprintf('\x%02X', \ord($c[$i]));
            } while (isset($c[++$i]));

            return $s . '</span>';
        }, $v) . '</span>';

        if (!($attr['binary'] ?? false)) {
            $v = preg_replace_callback(static::$unicodeCharsRx, function ($c) {
                return '<span class=sf-dump-default>\u{' . strtoupper(dechex(mb_ord($c[0]))) . '}</span>';
            }, $v);
        }

        if (isset($attr['file']) && $href = $this->getSourceLink($attr['file'], $attr['line'] ?? 0)) {
            $attr['href'] = $href;
        }
        if (isset($attr['href'])) {
            if ('label' === $style) {
                $v .= '^';
            }
            $target = isset($attr['file']) ? '' : ' target="_blank"';
            $v = sprintf(
                '<a href="%s"%s rel="noopener noreferrer">%s</a>',
                esc($this->utf8Encode($attr['href'])),
                $target,
                $v,
            );
        }
        if (isset($attr['lang'])) {
            $v = sprintf('<code class="%s">%s</code>', esc($attr['lang']), $v);
        }
        if ('label' === $style) {
            $v .= ' ';
        }

        return $v;
    }

    protected function dumpLine(int $depth, bool $endOfValue = false): void
    {
        if (-1 === $this->lastDepth) {
            $this->line = \sprintf($this->dumpPrefix, $this->dumpId, $this->indentPad) . $this->line;
        }

        if (-1 === $depth) {
            $args = ['"' . $this->dumpId . '"'];
            if ($this->extraDisplayOptions) {
                $args[] = \json_encode($this->extraDisplayOptions, \JSON_FORCE_OBJECT);
            }
            // Replace is for BC
            $this->line .= \sprintf(\str_replace('"%s"', '%s', $this->dumpSuffix), \implode(', ', $args));
        }
        $this->lastDepth = $depth;

        $this->line = \mb_encode_numericentity($this->line, [0x80, 0x10FFFF, 0, 0x1FFFFF], 'UTF-8');

        if (-1 === $depth) {
            AbstractDumper::dumpLine(0);
        }
        AbstractDumper::dumpLine($depth);
    }

    private function getSourceLink(string $file, int $line): string|false
    {
        $options = $this->extraDisplayOptions + $this->displayOptions;

        if ($fmt = $options['fileLinkFormat']) {
            return \is_string($fmt) ? strtr($fmt, ['%f' => $file, '%l' => $line]) : $fmt->format($file, $line);
        }

        return false;
    }
}

function esc(string $str): string
{
    return htmlspecialchars($str, \ENT_QUOTES, 'UTF-8');
}
