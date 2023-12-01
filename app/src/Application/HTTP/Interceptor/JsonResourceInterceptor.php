<?php

declare(strict_types=1);

namespace App\Application\HTTP\Interceptor;

use App\Application\HTTP\Response\ErrorResource;
use App\Application\HTTP\Response\ResourceInterface;
use App\Application\HTTP\Response\ValidationResource;
use Psr\Http\Message\ResponseFactoryInterface;
use Spiral\Core\CoreInterceptorInterface;
use Spiral\Core\CoreInterface;
use Spiral\Exceptions\ExceptionHandlerInterface;
use Spiral\Filters\Exception\ValidationException;

final class JsonResourceInterceptor implements CoreInterceptorInterface
{
    public function __construct(
        private readonly ResponseFactoryInterface $responseFactory,
        private readonly ExceptionHandlerInterface $exceptionHandler,
    ) {
    }

    public function process(string $controller, string $action, array $parameters, CoreInterface $core): mixed
    {
        try {
            $response = $core->callAction($controller, $action, $parameters);
        } catch (ValidationException $e) {
            $response = new ValidationResource($e);
        } catch (\Throwable $e) {
            $this->exceptionHandler->report($e);
            $response = new ErrorResource($e);
        }

        if ($response instanceof ResourceInterface) {
            $response = $response->toResponse(
                $this->responseFactory->createResponse(),
            );
        }

        return $response;
    }
}
