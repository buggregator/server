<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\DTO;

final class Payload implements \JsonSerializable
{
    public static function parse(string $data): self
    {
        return new self(
            \array_map(
                static fn(string $payload): PayloadChunkInterface => self::parsePayload($payload),
                \array_filter(\explode("\n", $data)),
            ),
        );
    }

    private static function parsePayload(string $payload): PayloadChunkInterface
    {
        if (\json_validate($payload)) {
            $json = \json_decode($payload, true, 512, \JSON_THROW_ON_ERROR);

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
    public function __construct(
        public array $chunks,
    ) {}

    public function getMeta(): PayloadChunkInterface
    {
        return $this->chunks[0];
    }

    public function getPayload(): PayloadChunkInterface
    {
        return $this->chunks[2];
    }

    public function type(): Type
    {
        foreach ($this->chunks as $chunk) {
            if ($chunk instanceof TypeChunk) {
                return $chunk->type();
            }
        }

        throw new \InvalidArgumentException('Type chunk not found');
    }

    public function jsonSerialize(): array
    {
        return $this->chunks;
    }
}
