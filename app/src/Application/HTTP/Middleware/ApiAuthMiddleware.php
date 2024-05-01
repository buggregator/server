<?php

declare(strict_types=1);

namespace App\Application\HTTP\Middleware;

use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;
use Spiral\Auth\Middleware\AuthTransportWithStorageMiddleware;
use Spiral\Core\FactoryInterface;

final class ApiAuthMiddleware implements MiddlewareInterface
{
    private ?MiddlewareInterface $middleware = null;

    public function __construct(
        private readonly FactoryInterface $factory,
    ) {}

    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler): ResponseInterface
    {
        if ($this->middleware === null) {
            $this->initMiddleware();
        }

        return $this->middleware->process($request, $handler);
    }

    private function initMiddleware(): void
    {
        $this->middleware = $this->factory->make(AuthTransportWithStorageMiddleware::class, [
            'transportName' => 'header',
        ]);
    }
}
