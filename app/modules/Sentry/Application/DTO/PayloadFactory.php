<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\DTO;

final class PayloadFactory
{
    public static function parseJson(string $json): Payload
    {
        $data = \array_filter(\explode("\n", $json));
        $chunks = [];
        foreach ($data as $i => $chunk) {
            $chunks[] = self::parsePayload($chunk, $i);
        }

        $platform = self::detectPlatform($chunks);

        return match ($platform) {
            Platform::React,
            Platform::Angular,
            Platform::Javascript => new JavascriptPayload($chunks),

            Platform::VueJs => new VueJsPayload($chunks),

            Platform::PHP,
            Platform::Laravel,
            Platform::Symfony => new PHPPayload($chunks),

            default => new Payload($chunks),
        };
    }

    private static function parsePayload(string $payload, int $index): PayloadChunkInterface
    {
        if (\json_validate($payload)) {
            $json = \json_decode($payload, true, 512, \JSON_THROW_ON_ERROR);

            if ($index === 0) {
                return new MetaChunk($json);
            }

            if (isset($json['type'])) {
                return new TypeChunk($json);
            }

            return new JsonChunk($json);
        }

        return new BlobChunk($payload);
    }

    /**
     * @param PayloadChunkInterface[] $chunks
     */
    private static function detectPlatform(array $chunks): Platform
    {
        foreach ($chunks as $chunk) {
            if ($chunk instanceof MetaChunk) {
                return $chunk->platform();
            }
        }

        throw new \InvalidArgumentException('Meta chunk not found');
    }
}
