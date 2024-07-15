<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\Handlers;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Event\EventType;
use App\Application\Event\StackStrategy;
use Modules\Sentry\Application\DTO\Payload;
use Modules\Sentry\Application\DTO\Type;
use Modules\Sentry\Application\EventHandlerInterface;
use Modules\Sentry\Application\Mapper\EventTypeMapper;
use Modules\Sentry\Domain\FingerprintRepositoryInterface;
use Spiral\Cqrs\CommandBusInterface;

final readonly class StoreEventHandler implements EventHandlerInterface
{
    public function __construct(
        private CommandBusInterface $commands,
        private FingerprintRepositoryInterface $fingerprints,
        private EventTypeMapper $mapper,
    ) {}

    public function handle(Payload $payload, EventType $event): Payload
    {
        if ($payload->type() === Type::Event) {
            // TODO: map event to preview
//            $data = $this->mapper->toPreview(
//                type: $event->type,
//                payload: [
//                    ...$payload->getPayload()->jsonSerialize(),
//                    'fingerprint' => $payload->getFingerprint(),
//                    'tags' => $payload->tags(),
//                ],
//            );

            $fingerprint = $payload->getFingerprint();
            $firstEvent = null;
            $lastEvent = null;
            $totalEvents = $this->fingerprints->totalEvents($fingerprint);

            if ($totalEvents === 1) {
                $firstEvent = $lastEvent = $this->fingerprints->findFirstSeen($fingerprint);
            } elseif ($totalEvents > 1) {
                $firstEvent = $this->fingerprints->findFirstSeen($fingerprint);
                $lastEvent = $this->fingerprints->findLastSeen($fingerprint);
            }

            $this->commands->dispatch(
                new HandleReceivedEvent(
                    type: $event->type,
                    payload: [
                        'tags' => $payload->tags(),
                        'total_events' => $totalEvents,
                        'first_event' => $firstEvent?->getCreatedAt(),
                        'last_event' => $lastEvent?->getCreatedAt(),
                        'fingerprint' => $fingerprint,
                        ...$payload->getPayload()->jsonSerialize(),
                    ],
                    project: $event->project,
//                    uuid: Uuid::fromString($this->md5ToUuid($payload->getFingerprint())),
                    groupId: $fingerprint,
                    stackStrategy: StackStrategy::OnlyLatest,
                ),
            );
        }

        return $payload;
    }

    private function md5ToUuid(string $hash): string
    {
        // Inserting hyphens to create a UUID format: 8-4-4-4-12
        return \substr($hash, 0, 8) . '-' .
            \substr($hash, 8, 4) . '-' .
            \substr($hash, 12, 4) . '-' .
            \substr($hash, 16, 4) . '-' .
            \substr($hash, 20, 12);
    }
}
