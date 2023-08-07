<?php

declare(strict_types=1);

namespace App\Interfaces\Http;

use App\Application\HTTP\Response\JsonResource;
use App\Application\HTTP\Response\ResourceInterface;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Router\Annotation\Route;

final class GetVersionAction
{
    #[Route(route: 'version', methods: ['GET'], group: 'api')]
    public function __invoke(EnvironmentInterface $env): ResourceInterface
    {
        return new JsonResource([
            'version' => $env->get('APP_VERSION', 'dev'),
        ]);
    }
}
