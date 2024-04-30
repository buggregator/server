<?php

declare(strict_types=1);

namespace Modules\Ray\Application\Handlers;

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
                            if (!\str_contains($val, 'Sfdump')) {
                                continue;
                            }
                            $event['payloads'][$i]['content'][$k][$j] = $this->cleanHtml($val);
                        }
                        continue;
                    }

                    if (\is_array($value) || !\str_contains($value, 'Sfdump')) {
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
        // Regex to find all instances of sf-dump- followed by digits
        $pattern = '/sf-dump-\d+/';
        // Perform the search
        \preg_match($pattern, $html, $matches);

        $sfDumpId = $matches[0] ?? null;
        // Remove everything except <pre> tags and their content
        return \preg_replace(
            '/(?s)(.*?)(<pre[^>]*>.*?<\/pre>)(.*)|(?s)(.*)/',
            '$2',
            $html,
        ) . '<script>Sfdump("' . $sfDumpId . '")</script>';
    }
}
