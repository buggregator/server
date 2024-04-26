<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Console\Command;

use Modules\Webhooks\Domain\WebhookLocatorInterface;
use Modules\Webhooks\Domain\WebhookRegistryInterface;
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
            $this->writeln("Registering webhook: {$webhook->uuid} for event: {$webhook->event} at {$webhook->url}");
            $registry->register($webhook);
        }

        return self::SUCCESS;
    }
}
