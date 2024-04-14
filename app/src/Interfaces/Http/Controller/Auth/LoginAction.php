<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Controller\Auth;

use App\Application\Auth\SuccessRedirect;
use Auth0\SDK\Auth0;
use Psr\Http\Message\ResponseInterface;
use Spiral\Auth\AuthScope;
use Spiral\Auth\TokenStorageInterface;
use Spiral\Http\ResponseWrapper;
use Spiral\Router\Annotation\Route;

final readonly class LoginAction
{
    public function __construct(
        private ResponseWrapper $response,
    ) {}

    #[Route(route: 'auth/sso/login', methods: ['GET'], group: 'guest')]
    public function __invoke(
        Auth0 $auth,
        AuthScope $authScope,
        TokenStorageInterface $tokens,
        SuccessRedirect $successRedirect,
    ): ResponseInterface {
        $session = $auth->getCredentials();

        if (null === $session || $session->accessTokenExpired) {
            return $this->response->redirect($auth->login());
        }

        $authScope->start(
            $token = $tokens->create($auth->getUser()),
        );

        return $successRedirect->makeResponse($token->getID());
    }
}
