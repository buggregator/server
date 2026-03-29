#!/usr/bin/env php
<?php

/**
 * VarDumper parser for Go-Buggregator.
 *
 * Reads base64-encoded PHP serialized VarDumper payloads from stdin (one per line),
 * deserializes them, converts to JSON, and writes to stdout (one JSON object per line).
 *
 * Protocol:
 *   IN:  base64(serialize([Data $data, array $context])) \n
 *   OUT: {"type":"string|array|object|...","value":"...","label":null,"context":{...}} \n
 *
 * On error:
 *   OUT: {"error":"message"} \n
 */

require __DIR__ . '/vendor/autoload.php';

use Symfony\Component\VarDumper\Cloner\Data;
use Symfony\Component\VarDumper\Cloner\Stub;
use Symfony\Component\VarDumper\Dumper\HtmlDumper;

// Disable output buffering for real-time line-by-line communication.
if (function_exists('ob_implicit_flush')) {
    ob_implicit_flush(true);
}

$dumpId = 0;

while (($line = fgets(STDIN)) !== false) {
    $line = trim($line);
    if ($line === '') {
        continue;
    }

    $result = processLine($line, $dumpId);
    $dumpId++;
    fwrite(STDOUT, json_encode($result, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . "\n");
}

function processLine(string $base64, int $dumpId): array
{
    // Decode base64.
    $raw = base64_decode($base64, true);
    if ($raw === false) {
        return ['error' => 'Invalid base64 encoding'];
    }

    // Unserialize with restricted classes.
    try {
        $payload = unserialize($raw, [
            'allowed_classes' => [Data::class, Stub::class],
        ]);
    } catch (\Throwable $e) {
        return ['error' => 'Unserialize failed: ' . $e->getMessage()];
    }

    if ($payload === false) {
        return ['error' => 'Unable to unserialize payload'];
    }

    // Validate structure: [Data, array].
    if (
        !is_array($payload)
        || count($payload) < 2
        || !$payload[0] instanceof Data
        || !is_array($payload[1])
    ) {
        return ['error' => 'Invalid payload structure'];
    }

    [$data, $context] = $payload;

    return convertData($data, $context, $dumpId);
}

function convertData(Data $data, array $context, int $dumpId): array
{
    $type = $data->getType();
    $dataContext = $data->getContext();

    // Determine label.
    $label = $dataContext['label'] ?? $context['label'] ?? null;

    // Check for code language override.
    $language = $dataContext['language'] ?? $context['language'] ?? null;

    // Extract project.
    $project = $dataContext['project'] ?? $context['project'] ?? null;

    // Convert value based on type.
    if (in_array($type, ['string', 'boolean', 'integer', 'double'], true)) {
        // Primitive: return raw value as string.
        $value = $data->getValue();
        $stringValue = match (true) {
            $value === true => '1',
            $value === false => '0',
            default => (string) $value,
        };

        $resultType = $type;
        if ($type === 'string' && $language !== null) {
            $resultType = 'code';
        }

        return [
            'type' => $resultType,
            'value' => $stringValue,
            'label' => $label,
            'language' => $language,
            'context' => $context,
            'project' => $project,
        ];
    }

    // Complex type: render as HTML using Symfony HtmlDumper.
    $dumper = new HtmlDumper();
    $dumper->setDumpHeader('');
    $dumper->setDumpBoundaries(
        '<pre class=sf-dump id=sf-dump-' . $dumpId . ' data-indent-pad="  ">',
        '</pre><script>Sfdump("sf-dump-' . $dumpId . '")</script>'
    );

    $html = '';
    $dumper->dump($data, function (string $line, int $depth) use (&$html) {
        if ($depth >= 0) {
            $html .= str_repeat('  ', $depth) . $line . "\n";
        } else {
            $html .= $line . "\n";
        }
    });

    return [
        'type' => $type,
        'value' => trim($html),
        'label' => $label,
        'language' => $language,
        'context' => $context,
        'project' => $project,
    ];
}
