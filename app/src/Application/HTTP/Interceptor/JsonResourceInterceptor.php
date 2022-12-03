<?php

declare(strict_types=1);

namespace App\Application\HTTP\Interceptor;

use App\Application\HTTP\Response\ResourceInterface;
use Psr\Http\Message\ResponseFactoryInterface;
use Spiral\Core\CoreInterceptorInterface;
use Spiral\Core\CoreInterface;
use Spiral\Http\Request\InputManager;

final class JsonResourceInterceptor implements CoreInterceptorInterface
{
    public function __construct(
        private readonly InputManager $manager,
        private readonly ResponseFactoryInterface $responseFactory
    ) {
    }

    public function process(string $controller, string $action, array $parameters, CoreInterface $core): mixed
    {
        $response = $core->callAction($controller, $action, $parameters);

        if ($response instanceof ResourceInterface) {
            $response = $response->toResponse(
                $this->manager->request(),
                $this->responseFactory->createResponse()
            );
        }

        return $response;
    }
}
