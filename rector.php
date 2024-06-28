<?php

declare(strict_types=1);

use Rector\CodingStyle\Rector\Catch_\CatchExceptionNameMatchingTypeRector;
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
    ->withCache(cacheDirectory: __DIR__ . '.cache/rector')
    ->withPhpVersion(PhpVersion::PHP_82)
    ->withPhpSets(php82: true)
    ->withPreparedSets(
        deadCode: true,
        codeQuality: true,
        codingStyle: true,
    )
    ->withImportNames(importShortClasses: false, removeUnusedImports: true)
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
    ->withSkip([
        CatchExceptionNameMatchingTypeRector::class,
    ]);
