<?php

declare(strict_types=1);

namespace Modules\Sms\Interfaces\Http\Handler;

use Modules\Sms\Application\Gateway\GatewayInterface;
use App\Application\Commands\HandleReceivedEvent;
use App\Application\Service\HttpHandler\HandlerInterface;
use Modules\Sms\Application\Gateway\GatewayRegistry;
use Modules\Sms\Domain\SmsMessage;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Core\Attribute\Singleton;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\ResponseWrapper;

#[Singleton]
final readonly class SmsHandler implements HandlerInterface
{
    public function __construct(
        private GatewayRegistry $registry,
        private CommandBusInterface $commands,
        private ResponseWrapper $responseWrapper,
    ) {}

    public function priority(): int
    {
        return 0;
    }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        $path = \rtrim($request->getUri()->getPath(), '/');

        if (!\str_starts_with($path, '/sms')) {
            return $next($request);
        }

        $body = $this->extractBody($request);

        if ($body === []) {
            return $next($request);
        }

        // Parse URL: /sms, /sms/{gateway}, /sms/{gateway}/{project}
        $parts = \explode('/', \ltrim($path, '/'));
        $segment1 = $parts[1] ?? null; // gateway name or project
        $segment2 = $parts[2] ?? null; // project (if gateway specified)

        // Try to find explicit gateway by URL segment
        $explicitGateway = $segment1 ? $this->registry->findByName($segment1) : null;

        if ($explicitGateway instanceof GatewayInterface) {
            // Explicit gateway via URL — validate and always store event
            $project = $segment2;
            $warnings = $explicitGateway->validate($body);
            $sms = $explicitGateway->parse($body);

            if ($warnings !== []) {
                $sms = new SmsMessage(
                    from: $sms->from,
                    to: $sms->to,
                    message: $sms->message,
                    gateway: $sms->gateway,
                    warnings: $warnings,
                );
            }

            $this->dispatchEvent($sms, $project);

            // Return error code if validation failed — the app sees the error,
            // but the event is still captured in Buggregator for inspection
            if ($warnings !== []) {
                return $this->responseWrapper->json([
                    'error' => 'validation_failed',
                    'gateway' => $explicitGateway->name(),
                    'missing_fields' => $warnings,
                ], 422);
            }

            return $this->responseWrapper->create(200);
        }

        // No explicit gateway — auto-detect
        $project = $segment1; // first segment is project in auto-detect mode
        $gateway = $this->registry->detect($body);

        if (!$gateway instanceof GatewayInterface) {
            return $next($request);
        }

        $sms = $gateway->parse($body);

        if ($sms->to === '' || $sms->message === '') {
            return $next($request);
        }

        $this->dispatchEvent($sms, $project);

        return $this->responseWrapper->create(200);
    }

    private function dispatchEvent(SmsMessage $sms, ?string $project): void
    {
        $this->commands->dispatch(
            new HandleReceivedEvent(
                type: 'sms',
                payload: $sms,
                project: $project,
            ),
        );
    }

    private function extractBody(ServerRequestInterface $request): array
    {
        $body = $request->getParsedBody();

        if (\is_array($body) && $body !== []) {
            return $body;
        }

        $json = \json_decode((string) $request->getBody(), true);

        return \is_array($json) ? $json : [];
    }
}
