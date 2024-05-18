<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use App\Application\Finder\Finder;
use App\Application\Persistence\DriverEnum;
use App\Interfaces\Console\RegisterModulesCommand;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\ORMInterface;
use Cycle\ORM\Select;
use GuzzleHttp\Client;
use GuzzleHttp\ClientInterface;
use Modules\Metrics\Application\CollectorRegistryInterface;
use Modules\Webhooks\Application\Locator\CompositeWebhookLocator;
use Modules\Webhooks\Application\Locator\WebhookFilesFinder;
use Modules\Webhooks\Application\Locator\WebhookFilesFinderInterface;
use Modules\Webhooks\Application\Locator\WebhookLocatorInterface;
use Modules\Webhooks\Application\Locator\WebhookRegistryInterface;
use Modules\Webhooks\Application\Locator\YamlFileWebhookLocator;
use Modules\Webhooks\Domain as Webhooks;
use Modules\Webhooks\Domain\DeliveryFactoryInterface;
use Modules\Webhooks\Domain\WebhookFactoryInterface;
use Modules\Webhooks\Domain\WebhookServiceInterface;
use Modules\Webhooks\Integartion\CycleOrm\WebhookRegistry;
use Modules\Webhooks\Integartion\CycleOrm\DeliveryRepository;
use Modules\Webhooks\Integartion\CycleOrm\WebhookRepository;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\DirectoriesInterface;
use Spiral\Console\Bootloader\ConsoleBootloader;
use Spiral\Core\FactoryInterface;
use Spiral\Queue\QueueConnectionProviderInterface;
use Spiral\RoadRunner\Metrics\Collector;

final class WebhooksBootloader extends Bootloader
{
    private const CACHE_ALIAS = 'webhooks';
    private const QUEUE_ALIAS = 'webhooks';

    public function defineSingletons(): array
    {
        return [
            ClientInterface::class => static fn(
                HttpClientSettings $settings,
            ): ClientInterface => new Client([
                'timeout' => $settings->getTimeout(),
                'connect_timeout' => 5,
                'headers' => [
                    'User-Agent' => $settings->getUserAgent(),
                    'Content-Type' => $settings->getContentType(),
                ],
            ]),

            WebhookFactoryInterface::class => WebhookFactory::class,
            WebhookServiceInterface::class => static fn(
                FactoryInterface $factory,
                QueueConnectionProviderInterface $provider,
            ): WebhookServiceInterface => $factory->make(
                WebhookService::class,
                [
                    'queue' => $provider->getConnection(self::QUEUE_ALIAS),
                ],
            ),


            YamlFileWebhookLocator::class => static fn(
                FactoryInterface $factory,
                DirectoriesInterface $dirs,
            ): YamlFileWebhookLocator => $factory->make(YamlFileWebhookLocator::class),

            WebhookFilesFinderInterface::class => static fn(
                FactoryInterface $factory,
                DirectoriesInterface $dirs,
            ): WebhookFilesFinderInterface => $factory->make(
                WebhookFilesFinder::class,
                [
                    'finder' => new Finder(
                        finder: \Symfony\Component\Finder\Finder::create()
                            ->files()
                            ->in($dirs->get('runtime') . '/configs')
                            ->name('*.webhook.yaml'),
                    ),
                ],
            ),

            WebhookLocatorInterface::class => static function (
                YamlFileWebhookLocator $locator,
            ): WebhookLocatorInterface {
                return new CompositeWebhookLocator([
                    $locator,
                ]);
            },

            DeliveryFactoryInterface::class => DeliveryFactory::class,

            // Persistence
            WebhookRepository::class => static fn(
                ORMInterface $orm,
                EntityManagerInterface $manager,
            ): Webhooks\WebhookRepositoryInterface => new WebhookRepository(
                $manager,
                new Select($orm, Webhooks\Webhook::class),
            ),
            DeliveryRepository::class => static fn(
                ORMInterface $orm,
                EntityManagerInterface $manager,
            ): Webhooks\DeliveryRepositoryInterface => new DeliveryRepository(
                $manager,
                new Select($orm, Webhooks\Delivery::class),
            ),
            Webhooks\WebhookRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                DriverEnum $driver,
            ): Webhooks\WebhookRepositoryInterface => match ($driver) {
                DriverEnum::Database => $factory->make(WebhookRepository::class),
                default => throw new \RuntimeException('Unsupported driver'),
            },

            Webhooks\DeliveryRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                DriverEnum $driver,
            ): Webhooks\DeliveryRepositoryInterface => match ($driver) {
                DriverEnum::Database => $factory->make(DeliveryRepository::class),
                default => throw new \RuntimeException('Unsupported driver'),
            },

            WebhookRegistryInterface::class => WebhookRegistry::class,
        ];
    }

    public function init(
        ConsoleBootloader $console,
        CollectorRegistryInterface $collectorRegistry,
    ): void {
        $console->addSequence(
            name: RegisterModulesCommand::SEQUENCE,
            sequence: 'webhooks:register',
            header: 'Register webhooks from configuration',
        );

        $collectorRegistry->register(
            name: 'webhooks',
            collector: Collector::counter()
                ->withHelp('Webhooks counter')
                ->withLabels('event', 'url', 'success'),
        );
    }
}
