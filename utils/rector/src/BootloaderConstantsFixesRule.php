<?php

namespace Utils\Rector;

use Symplify\RuleDocGenerator\ValueObject\CodeSample\CodeSample;
use PhpParser\Node;
use PhpParser\Node\Stmt\Class_;
use PhpParser\Node\Stmt\ClassConst;
use PhpParser\Node\Stmt\ClassMethod;
use PhpParser\Node\Stmt\Return_;
use PhpParser\Node\Expr\Array_;
use PhpParser\Node\Expr\ArrayItem;
use PHPStan\Type\ObjectType;
use Rector\Rector\AbstractRector;
use Symplify\RuleDocGenerator\ValueObject\RuleDefinition;

final class BootloaderConstantsFixesRule extends AbstractRector
{
    public function getRuleDefinition(): RuleDefinition
    {
        return new RuleDefinition(
            description: 'Change bootloader const to method call',
            codeSamples: [
                new CodeSample(
                    <<<PHP
final class AttributesBootloader extends Bootloader
{
    protected const DEPENDENCIES = [
        SomeBootloader::class,
    ];
}
PHP
                    ,
                    <<<PHP
final class AttributesBootloader extends Bootloader
{
    public function defineDependencies(): array
    {
        return [
            SomeBootloader::class,
        ];
    }
}
PHP,
                ),
                new CodeSample(
                    <<<PHP
final class AttributesBootloader extends Bootloader
{
    protected const SINGLETONS = [
        SomeClassInterface::class => SomeClass::class,
    ];
}
PHP
                    ,
                    <<<PHP
final class AttributesBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            SomeClassInterface::class => SomeClass::class,
        ];
    }
}
PHP,
                ),
                new CodeSample(
                    <<<PHP
final class AttributesBootloader extends Bootloader
{
    protected const BINDINGS = [
        SomeClassInterface::class => SomeClass::class,
    ];
}
PHP
                    ,
                    <<<PHP
final class AttributesBootloader extends Bootloader
{
    public function defineBindings(): array
    {
        return [
            SomeClassInterface::class => SomeClass::class,
        ];
    }
}
PHP,
                ),
            ],
        );
    }

    public function getNodeTypes(): array
    {
        return [Class_::class];
    }


    public function refactor(Node $node): ?Node
    {
        if (!$node instanceof Class_ || $node->extends === null) {
            return null;
        }

        // Check if the class extends Spiral\Boot\Bootloader\Bootloader
        $parentClass = $node->extends;
        if (!$this->isObjectType($parentClass, new ObjectType('Spiral\Boot\Bootloader\Bootloader'))) {
            return null;
        }

        $methodsToAdd = [];
        foreach ($node->stmts as $key => $stmt) {
            if ($stmt instanceof ClassConst) {
                $constName = $stmt->consts[0]->name->toString();
                $methodName = match ($constName) {
                    'DEPENDENCIES' => 'defineDependencies',
                    'SINGLETONS' => 'defineSingletons',
                    'BINDINGS' => 'defineBindings',
                    default => null,
                };

                if ($methodName === null) {
                    continue;
                }

                $return = $constName === 'DEPENDENCIES'
                    ? $stmt->consts[0]->value
                    : $this->convertBindingsArray($stmt->consts[0]->value, $node);

                $methodsToAdd[] = [
                    'key' => $key,
                    'method' => new ClassMethod($methodName, [
                        'flags' => Class_::MODIFIER_PUBLIC,
                        'returnType' => 'array',
                        'stmts' => [
                            new Return_($return),
                        ],
                    ]),
                ];
            }
        }

        foreach ($methodsToAdd as $methodToAdd) {
            $node->stmts[$methodToAdd['key']] = $methodToAdd['method'];
        }

        return $node;
    }

    private function convertBindingsArray(Array_ $oldSingletons, Class_ $class): Array_
    {
        $items = [];
        foreach ($oldSingletons->items as $item) {
            $classType = $item->key;
            $value = $item->value;

            if ($value instanceof Array_) { // Method reference
                $methodName = $value->items[1]->value;
                $methodNode = $this->findMethod($class, $methodName->value);
                if ($methodNode !== null) {
                    $params = $methodNode->params;
                    $returnType = $methodNode->returnType;
                    $body = $methodNode->stmts;

                    $arrowFunction = new Node\Expr\ArrowFunction([
                        'params' => $params,
                        'expr' => $body[0]->expr,
                        'static' => true,
                        'returnType' => $returnType,
                    ]);

                    $items[] = new ArrayItem($arrowFunction, $classType);
                }
            } else { // Direct class reference
                $items[] = new ArrayItem($value, $classType);
            }
        }

        return new Array_($items);
    }

    private function findMethod(Class_ $class, string $methodName): ?ClassMethod
    {
        foreach ($class->stmts as $stmt) {
            if ($stmt instanceof ClassMethod && $stmt->name->toString() === $methodName) {
                return $stmt;
            }
        }
        return null;
    }
}
