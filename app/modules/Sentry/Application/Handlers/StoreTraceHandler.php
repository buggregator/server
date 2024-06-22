<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\Handlers;

use App\Application\Event\EventType;
use Cycle\ORM\EntityManagerInterface;
use Modules\Sentry\Application\DTO\Exception;
use Modules\Sentry\Application\DTO\JsonChunk;
use Modules\Sentry\Application\DTO\Payload;
use Modules\Sentry\Application\DTO\Type;
use Modules\Sentry\Application\EventHandlerInterface;
use Modules\Sentry\Domain\FingerprintFactoryInterface;
use Modules\Sentry\Domain\Issue;
use Modules\Sentry\Domain\IssueFactoryInterface;
use Modules\Sentry\Domain\IssueTag;
use Modules\Sentry\Domain\Trace;
use Modules\Sentry\Domain\TraceFactoryInterface;
use Modules\Sentry\Domain\TraceRepositoryInterface;

final readonly class StoreTraceHandler implements EventHandlerInterface
{
    public function __construct(
        private TraceRepositoryInterface $traces,
        private TraceFactoryInterface $traceFactory,
        private IssueFactoryInterface $issueFactory,
        private FingerprintFactoryInterface $fingerprintFactory,
        private EntityManagerInterface $em,
    ) {}

    public function handle(Payload $payload, EventType $event): Payload
    {
        $trace = $this->findOrCreateTrace($payload);

        return match ($payload->type()) {
            Type::Event => $this->storeEvent($payload, $trace),
            Type::Transaction => $this->storeTransaction($payload, $trace),
            default => $payload,
        };
    }

    private function findOrCreateTrace(Payload $payload): Trace
    {
        $trace = $this->traces->findOne(['trace_id' => $payload->traceId()]);
        if (!$trace) {
            $trace = $this->traceFactory->createFromMeta(
                uuid: $payload->uuid,
                meta: $payload->getMeta(),
            );
            $this->em->persist($trace)->run();
        }

        return $trace;
    }

    private function storeEvent(Payload $payload, Trace $trace): Payload
    {
        $json = $payload->getPayload();
        \assert($json instanceof JsonChunk);

        $issue = $this->issueFactory->createFromPayload(
            traceUuid: $trace->getUuid(),
            payload: $json,
        );

        $this->em->persist($issue);

        $exceptions = \array_map(
            static fn(array $exception) => new Exception($exception),
            (array) ($json['exception']['values'] ?? []),
        );

        $fingerprint = $this->fingerprintFactory->create(
            issueUuid: $issue->getUuid(),
            exceptions: $exceptions,
        );

        $this->em->persist($fingerprint);
        $this->storeTags($payload, $issue);
        $this->em->run();

        return $payload->withFingerprint($fingerprint->getFingerprint());
    }

    private function storeTransaction(Payload $payload, Trace $trace): Payload
    {
        // todo: implement

        return $payload;
    }

    private function storeTags(Payload $payload, Issue $issue): void
    {
        foreach ($payload->tags() as $tag => $value) {
            $issue->getTags()->add(
                new IssueTag(
                    issueUuid: $issue->getUuid(),
                    tag: $tag,
                    value: (string) $value,
                ),
            );
        }
    }
}
