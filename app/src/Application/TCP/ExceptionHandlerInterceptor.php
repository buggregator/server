<?php

declare(strict_types=1);

namespace App\Application\TCP;

use Spiral\Core\CoreInterceptorInterface;
use Spiral\Core\CoreInterface;
use Spiral\Exceptions\ExceptionReporterInterface;
use Spiral\RoadRunnerBridge\Tcp\Response\CloseConnection;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;

final readonly class ExceptionHandlerInterceptor implements CoreInterceptorInterface
{
    public function __construct(
        private ExceptionReporterInterface $reporter,
    ) {}

    public function process(string $controller, string $action, array $parameters, CoreInterface $core): ResponseInterface
    {
        try {
            return $core->callAction($controller, $action, $parameters);
        } catch (\Throwable $e) {
            $this->reporter->report($e);
            return new CloseConnection();
        }
    }
}
