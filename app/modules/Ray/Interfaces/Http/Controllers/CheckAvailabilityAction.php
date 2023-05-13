<?php

declare(strict_types=1);

namespace Modules\Ray\Interfaces\Http\Controllers;

use Psr\Http\Message\ResponseInterface;
use Spiral\Http\ResponseWrapper;
use Spiral\Router\Annotation\Route;

final class CheckAvailabilityAction
{
    public function __construct(
        private readonly ResponseWrapper $response,
    ) {
    }

    #[Route(route: '/_availability_check', name: 'ray.availability_check')]
    public function __invoke(): ResponseInterface
    {
        return $this->response->create(400);
    }
}
