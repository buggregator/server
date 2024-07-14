<?php

declare(strict_types=1);

use Rector\Config\RectorConfig;
use Rector\Php74\Rector\Closure\ClosureToArrowFunctionRector;
use Rector\Php80\Rector\Catch_\RemoveUnusedVariableInCatchRector;
use Rector\Php80\Rector\FuncCall\ClassOnObjectRector;
use Rector\Php80\Rector\FunctionLike\MixedTypeRector;
use Rector\Php80\Rector\Identical\StrEndsWithRector;
use Rector\Php80\Rector\Identical\StrStartsWithRector;
use Rector\Php80\Rector\NotIdentical\StrContainsRector;
use Rector\Php82\Rector\Class_\ReadOnlyClassRector;
use Rector\Php82\Rector\Param\AddSensitiveParameterAttributeRector;
use Rector\TypeDeclaration\Rector\ClassMethod\AddVoidReturnTypeWhereNoReturnRector;
use Rector\ValueObject\PhpVersion;
use Utils\Rector\BootloaderConstantsFixesRule;

return RectorConfig::configure()
    ->withPaths([
        __DIR__ . '/app',
        __DIR__ . '/tests',
    ])
    ->withImportNames(importShortClasses: false, removeUnusedImports: true)
    ->withCache(cacheDirectory: __DIR__ . '.cache/rector')
    // uncomment to reach your current PHP version
    // ->withPhpSets()
    ->withRules([
        BootloaderConstantsFixesRule::class,
        AddVoidReturnTypeWhereNoReturnRector::class,
        ReadOnlyClassRector::class,
        AddSensitiveParameterAttributeRector::class,
        ClassOnObjectRector::class,
        MixedTypeRector::class,
        StrEndsWithRector::class,
        StrStartsWithRector::class,
        StrContainsRector::class,
        RemoveUnusedVariableInCatchRector::class,
        ClosureToArrowFunctionRector::class,
    ])
    ->withPhpVersion(PhpVersion::PHP_82)
    // here we can define, what prepared sets of rules will be applied
    ->withPreparedSets(
        deadCode: true,
        codeQuality: true,
    );
