<?php

declare(strict_types=1);

namespace Modules\Sentry\Interfaces\Http\Controllers;

use App\Application\HTTP\GzippedStreamFactory;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Router\Annotation\Route;

final class EnvelopeAction
{
    public function __construct(
        private readonly GzippedStreamFactory $gzippedStreamFactory,
    ) {
    }

    #[Route(route: '<projectId>/envelope', name: 'sentry.event.envelope', methods: ['POST'], group: 'api')]
    public function __invoke(int $projectId, ServerRequestInterface $request): void
    {
        $data = $this->gzippedStreamFactory->createFromRequest($request)->getEnvelopePayload();

        if (\count($data) == 3) {
            match ($data[1]['type']) {
                'transaction' => $this->handleTransaction($data),
                'session' => $this->handleSession($data),
            };
        }
    }

    private function handleTransaction(array $data): void
    {
        // TODO handle sentry transaction
    }

    private function handleSession(array $data): void
    {
        // TODO handle sentry session
    }
}
