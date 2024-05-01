<?php

declare(strict_types=1);

namespace Modules\Ray\Application\Handlers;

use Modules\Ray\Application\DumpIdParser;
use Modules\Sentry\Application\EventHandlerInterface;

final class RemoveSfDumpScriptHandler implements EventHandlerInterface
{
    public function handle(array $event): array
    {
        if (!isset($event['payloads']) || !\is_array($event['payloads'])) {
            return $event;
        }

        foreach ($event['payloads'] as $i => $payload) {
            if (isset($payload['content']) && \is_array($payload['content'])) {
                foreach ($payload['content'] as $k => $value) {
                    if ($k === 'values') {
                        foreach ($value as $j => $val) {
                            if (!\is_string($val) || !\str_contains($val, 'Sfdump')) {
                                continue;
                            }
                            $event['payloads'][$i]['content'][$k][$j] = $this->cleanHtml($val);
                        }
                        continue;
                    }

                    if (\is_array($value) || !\is_string($value) || !\str_contains($value, 'Sfdump')) {
                        continue;
                    }
                    $event['payloads'][$i]['content'][$k] = $this->cleanHtml($value);
                }
            }
        }

        return $event;
    }

    private function cleanHtml(string $html): string
    {
        $sfDumpId = DumpIdParser::find($html);

        // Remove everything except <pre> tags and their content
        return \preg_replace(
            '/(?s)(.*?)(<pre[^>]*>.*?<\/pre>)(.*)|(?s)(.*)/',
            '$2',
            $html,
        ) . '<script>Sfdump("' . $sfDumpId . '")</script>';
    }
}
