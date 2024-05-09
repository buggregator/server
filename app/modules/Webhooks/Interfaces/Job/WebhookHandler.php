<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Job;

use App\Application\Domain\ValueObjects\Uuid;
use GuzzleHttp\ClientInterface;
use GuzzleHttp\Exception\RequestException;
use Modules\Webhooks\Application\WebhookMetrics;
use Modules\Webhooks\Domain\DeliveryFactoryInterface;
use Modules\Webhooks\Domain\DeliveryRepositoryInterface;
use Modules\Webhooks\Domain\Webhook;
use Modules\Webhooks\Domain\WebhookRepositoryInterface;
use Psr\Log\LoggerInterface;
use Spiral\Core\InvokerInterface;
use Spiral\Exceptions\ExceptionReporterInterface;
use Spiral\Queue\JobHandler;

final class WebhookHandler extends JobHandler
{
    public function __construct(
        InvokerInterface $invoker,
        private readonly WebhookRepositoryInterface $webhooks,
        private readonly DeliveryRepositoryInterface $deliveries,
        private readonly DeliveryFactoryInterface $deliveryFactory,
        private readonly ClientInterface $httpClient,
        private readonly ExceptionReporterInterface $reporter,
        private readonly LoggerInterface $logger,
        private readonly WebhookMetrics $metrics,
        private readonly RetryPolicy $retryPolicy,
    ) {
        parent::__construct($invoker);
    }

    public function invoke(JobPayload $payload): void
    {
        $uuid = Uuid::fromString($payload->webhookUuid->toString());

        $webhook = $this->webhooks->getByUuid($uuid);

        if (!$webhook->retryOnFailure) {
            $this->retryPolicy->setMaxRetries(1);
        }

        while ($this->retryPolicy->canRetry()) {
            try {
                $this->send($webhook, $payload);
                $this->logger->debug('Webhook sent', ['webhook' => (string) $webhook->uuid]);
                break;
            } catch (\Throwable) {
                $this->retryPolicy->nextRetry();
            }
        }
    }

    private function send(Webhook $webhook, JobPayload $payload): void
    {
        $failed = false;
        try {
            $headers = [
                'Content-Type' => 'application/json',
                'X-Webhook-Id' => (string) $webhook->uuid,
                'X-Webhook-Event' => $payload->event->event,
            ];

            foreach ($webhook->getHeaders() as $header => $value) {
                $headers[$header] = $webhook->getHeaderLine($header);
            }

            $response = $this->httpClient->request('POST', (string) $webhook->url, [
                'json' => $payload->event->payload,
                'verify' => $webhook->verifySsl,
                'headers' => $headers,
            ]);
        } catch (RequestException $e) {
            $this->reporter->report($e);
            $response = $e->getResponse();
            $failed = true;
            // Handle exception
        } finally {
            $delivery = $this->deliveryFactory->create(
                webhookUuid: $webhook->uuid,
                payload: \json_encode($payload->event->payload),
                response: (string) $response->getBody()->getContents(),
                status: $response?->getStatusCode() ?? 500,
            );

            $this->deliveries->store($delivery);
            $this->metrics->called($payload->event->event, (string) $webhook->url, !$failed);
        }

        if ($failed) {
            throw new \RuntimeException('Failed to deliver webhook');
        }
    }
}
