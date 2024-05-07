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
                $this->writeln("Registering webhook: {$webhook->key} for event: {$webhook->event} at {$webhook->url}");
                $registry->register($webhook);
            } catch (WebhooksAlreadyExistsException) {
                $this->warning("Webhook with key {$webhook->key} already exists");
            }
        }

        return self::SUCCESS;
    }
}
