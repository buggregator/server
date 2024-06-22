<?php

declare(strict_types=1);

namespace Modules\Sentry\Interfaces\Http;

use App\Application\Exception\EntityNotFoundException;
use App\Application\HTTP\Response\JsonResource;
use App\Application\HTTP\Response\ResourceInterface;
use Modules\Sentry\Domain\IssueRepositoryInterface;
use Spiral\Router\Annotation\Route;

final readonly class ShowIssueAction
{
    public function __construct(
        private IssueRepositoryInterface $issues,
    ) {}

    #[Route(route: '/sentry/issue/<fingerprint>', name: 'sentry.latest_issue', methods: 'GET', group: 'api')]
    public function __invoke(string $fingerprint): ResourceInterface
    {
        $issue = $this->issues->findLatestByFingerprint($fingerprint);

        if (!$issue) {
            throw new EntityNotFoundException('Issue not found');
        }

        return new JsonResource([
            'uuid' => (string) $issue->getUuid(),
            'title' => $issue->getTitle(),
            'platform' => $issue->getPlatform(),
            'logger' => $issue->getLogger(),
            'type' => $issue->getType(),
            'transaction' => $issue->getTransaction(),
            ...$issue->getPayload()->jsonSerialize(),
        ]);
    }
}
