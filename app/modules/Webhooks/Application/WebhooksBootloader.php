<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use GuzzleHttp\ClientInterface;
use Modules\Webhooks\Domain\WebhookFactoryInterface;
use Modules\Webhooks\Domain\WebhookLocatorInterface;
use Modules\Webhooks\Domain\WebhookRegistryInterface;
use Modules\Webhooks\Domain\WebhookRepositoryInterface;
use Modules\Webhooks\Domain\WebhookServiceInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\DirectoriesInterface;
use Spiral\Core\FactoryInterface;

final class WebhooksBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            ClientInterface::class => static fn(
            ): ClientInterface => new \GuzzleHttp\Client([
                'timeout' => 5,
                'connect_timeout' => 5,
                'headers' => [
                    'User-Agent' => 'Buggregator/webhooks',
                    'Content-Type' => 'application/json',
                ],
            ]),

            WebhookFactoryInterface::class => WebhookFactory::class,

            WebhookServiceInterface::class => WebhookService::class,

            InMemoryWebhookRepository::class => InMemoryWebhookRepository::class,
            WebhookRepositoryInterface::class => InMemoryWebhookRepository::class,
            WebhookRegistryInterface::class => InMemoryWebhookRepository::class,

            YamlFileWebhookLocator::class => static fn(
                FactoryInterface $factory,
                DirectoriesInterface $dirs,
            ): YamlFileWebhookLocator => $factory->make(
                YamlFileWebhookLocator::class,
                [
                    'directory' => $dirs->get('runtime') . '/webhooks',
                ],
            ),

            WebhookLocatorInterface::class => static function (
                YamlFileWebhookLocator $locator,
            ): WebhookLocatorInterface {
                return new CompositeWebhookLocator([
                    $locator,
                ]);
            },
        ];
    }

    public function init(
        WebhookRegistryInterface $registry,
        WebhookLocatorInterface $locator,
    ): void {
        foreach ($locator->findAll() as $webhook) {
            $registry->register($webhook);
        }
    }
}
