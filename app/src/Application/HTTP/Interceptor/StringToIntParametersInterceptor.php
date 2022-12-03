<?php

declare(strict_types=1);

namespace App\Application\HTTP\Interceptor;

use Spiral\Core\CoreInterceptorInterface;
use Spiral\Core\CoreInterface;

final class StringToIntParametersInterceptor implements CoreInterceptorInterface
{
    public function process(string $controller, string $action, array $parameters, CoreInterface $core): mixed
    {
        foreach ($parameters as $key => $parameter) {
            if (ctype_digit($parameter)) {
                $parameters[$key] = (int)$parameter;
            }
        }

        return $core->callAction($controller, $action, $parameters);
    }
}
