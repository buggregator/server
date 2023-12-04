<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use Doctrine\Common\Annotations\AnnotationReader;
use Spiral\Attributes\AttributeReader;
use Spiral\Attributes\Internal\Instantiator\NamedArgumentsInstantiator;
use Spiral\Attributes\ReaderInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Core\BinderInterface;

final class AttributesBootloader extends Bootloader
{
    protected const DEPENDENCIES = [
        \Spiral\Bootloader\Attributes\AttributesBootloader::class,
    ];

    public function init(BinderInterface $binder): void
    {
        AnnotationReader::addGlobalIgnoredName('mixin');

        $binder->bindSingleton(
            ReaderInterface::class,
            static fn(): ReaderInterface => new AttributeReader(
                new NamedArgumentsInstantiator(),
            ),
        );
    }
}
