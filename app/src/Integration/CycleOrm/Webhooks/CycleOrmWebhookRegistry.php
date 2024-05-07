<?php

declare(strict_types=1);

namespace App\Integration\CycleOrm\Webhooks;

use App\Application\Domain\Entity\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Modules\Webhooks\Application\Locator\Webhook;
use Modules\Webhooks\Application\Locator\WebhookRegistryInterface;
use Modules\Webhooks\Domain\ValueObject\Url;
use Modules\Webhooks\Domain\WebhookFactoryInterface;
use Modules\Webhooks\Domain\WebhookRepositoryInterface;
use Modules\Webhooks\Exceptions\WebhooksAlreadyExistsException;

final readonly class CycleOrmWebhookRegistry implements WebhookRegistryInterface
{
    public function __construct(
        private WebhookRepositoryInterface $webhooks,
        private WebhookFactoryInterface $factory,
    ) {}

    public function register(Webhook $webhook): Uuid
    {
        if ($this->webhooks->findByKey($webhook->key) !== null) {
            throw new WebhooksAlreadyExistsException(
                \sprintf('Webhook with key %s already exists', $webhook->key),
            );
        }

        $entity = $this->factory->create(
            key: $webhook->key,
            event: $webhook->event,
            url: Url::create($webhook->url),
            headers: new Json($webhook->headers),
            verifySsl: $webhook->verifySsl,
            retryOnFailure: $webhook->retryOnFailure,
        );

        $this->webhooks->store($entity);

        return $entity->getUuid();
    }
}
