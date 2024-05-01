<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Controller\Auth;

use App\Application\OAuth\AuthProviderInterface;
use Psr\Http\Message\ResponseInterface;
use Spiral\Http\ResponseWrapper;
use Spiral\Router\Annotation\Route;

final readonly class LogoutAction
{
    public function __construct(
        private ResponseWrapper $response,
    ) {}

    #[Route(route: 'auth/sso/logout', methods: ['GET'], group: 'guest')]
    public function __invoke(
        AuthProviderInterface $auth,
    ): ResponseInterface {

        $auth->logout();

        return $this->response->create(200);
    }
}
