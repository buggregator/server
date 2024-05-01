<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application;

use App\Application\Event\EventTypeRegistryInterface;
use Modules\VarDumper\Application\Dump\DumpIdGeneratorInterface;
use Modules\VarDumper\Application\Dump\MtRandDumpIdGenerator;
use Modules\VarDumper\Interfaces\TCP\Service;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Cqrs\CommandBusInterface;

final class VarDumperBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            DumpIdGeneratorInterface::class => MtRandDumpIdGenerator::class,
            Service::class => static fn(
                CommandBusInterface $bus,
                DumpIdGeneratorInterface $idGenerator,
                EnvironmentInterface $env,
            ): Service => new Service(
                bus: $bus,
                idGenerator: $idGenerator,
                previewMaxDepth: (int) $env->get('VAR_DUMPER_PREVIEW_MAX_DEPTH', 3),
            ),
        ];
    }

    public function boot(
        EventTypeRegistryInterface $registry,
        Mapper\EventTypeMapper $mapper,
    ): void {
        $registry->register('var-dump', $mapper);
    }
}
