<?php

declare(strict_types=1);

namespace App\Application\HTTP\Interceptor;

use App\Application\Domain\ValueObjects\Uuid;
use Spiral\Core\CoreInterceptorInterface;
use Spiral\Core\CoreInterface;

final class UuidParametersConverterInterceptor implements CoreInterceptorInterface
{
    public function process(string $controller, string $action, array $parameters, CoreInterface $core): mixed
    {
        $refMethod = new \ReflectionMethod($controller, $action);

        foreach ($refMethod->getParameters() as $parameter) {
            if ($parameter->getType() === Uuid::class) {
                $parameters[$parameter->getName()] = Uuid::fromString($parameters[$parameter->getName()]);
            }
        }

        return $core->callAction($controller, $action, $parameters);
    }
}
