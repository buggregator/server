<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\DTO;

final readonly class TypeChunk extends JsonChunk
{
    public function type(): Type
    {
        return match ($this->data['type']) {
            'event' => Type::Event,
            'span' => Type::Span,
            'transaction' => Type::Transaction,
            'replay_event' => Type::ReplyEvent,
            'replay_recording' => Type::ReplayRecording,
            default => throw new \InvalidArgumentException('Invalid type: ' . $this->data['type']),
        };
    }
}
