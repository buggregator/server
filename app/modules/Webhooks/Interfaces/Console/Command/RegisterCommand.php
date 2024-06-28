<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Console\Command;

use Modules\Webhooks\Application\Locator\WebhookLocatorInterface;
use Modules\Webhooks\Application\Locator\WebhookRegistryInterface;
use Modules\Webhooks\Exceptions\WebhooksAlreadyExistsException;
use Spiral\Console\Attribute\AsCommand;
use Spiral\Console\Command;

#[AsCommand(
    name: 'webhooks:register',
    description: 'Register webhooks from configuration',
)]
final class RegisterCommand extends Command
{
    public function __invoke(
        WebhookRegistryInterface $registry,
        WebhookLocatorInterface $locator,
    ): int {
        foreach ($locator->findAll() as $webhook) {
            try {
                $this->writeln(sprintf('Registering webhook: %s for event: %s at %s', $webhook->key, $webhook->event, $webhook->url));
                $registry->register($webhook);
            } catch (WebhooksAlreadyExistsException) {
                $this->warning(sprintf('Webhook with key %s already exists', $webhook->key));
            }
        }

        return self::SUCCESS;
    }
}
