<?php

declare(strict_types=1);

namespace Modules\Projects\Application;

use App\Application\Mode;
use Modules\Projects\Domain\ProjectFactoryInterface;
use Modules\Projects\Domain\ProjectLocatorInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\DirectoriesInterface;
use Spiral\Console\Bootloader\ConsoleBootloader;
use Spiral\Core\FactoryInterface;

final class ProjectBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            ProjectFactoryInterface::class => ProjectFactory::class,

            YamlFileProjectLocator::class => static fn(
                FactoryInterface $factory,
                DirectoriesInterface $dirs,
            ): YamlFileProjectLocator => $factory->make(
                YamlFileProjectLocator::class,
                [
                    'directory' => $dirs->get('runtime') . '/configs',
                ],
            ),

            ProjectLocatorInterface::class => static function (
                YamlFileProjectLocator $locator,
            ): ProjectLocatorInterface {
                return new CompositeProjectLocator([
                    $locator,
                ]);
            },

        ];
    }

    public function init(ConsoleBootloader $console, Mode $mode): void
    {
        if ($mode->insideRoadRunner()) {
            $console->addConfigureSequence(
                sequence: 'projects:register',
                header: 'Register all projects in the system',
            );
        }
    }
}
