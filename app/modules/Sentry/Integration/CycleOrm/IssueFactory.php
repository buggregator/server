<?php

declare(strict_types=1);

namespace Modules\Sentry\Integration\CycleOrm;

use App\Application\Domain\ValueObjects\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Modules\Sentry\Application\DTO\JsonChunk;
use Modules\Sentry\Domain\Issue;
use Modules\Sentry\Domain\IssueFactoryInterface;

final class IssueFactory implements IssueFactoryInterface
{
    public function createFromPayload(
        Uuid $traceUuid,
        JsonChunk $payload,
    ): Issue {
        return new Issue(
            uuid: Uuid::generate(),
            traceUuid: $traceUuid,
            title: $this->generateTitle($payload),
            platform: $payload['platform'] ?? 'unknown',
            logger: $payload['logger'] ?? 'unknown',
            type: 'error',
            transaction: $payload['transaction'] ?? null,
            serverName: $payload['server_name'] ?? '',
            payload: new Json($payload),
        );
    }

    private function generateTitle(JsonChunk $payload): string
    {
        $title = 'Unknown error';
        $exceptions = \array_reverse((array) ($payload['exception']['values'] ?? []));

        foreach ($exceptions as $exception) {
            if (isset($exception['value'])) {
                return $exception['value'];
            }
        }

        return $title;
    }
}
