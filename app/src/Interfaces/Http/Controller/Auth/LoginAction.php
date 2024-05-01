<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Controller\Auth;

use App\Application\Auth\SuccessRedirect;
use App\Application\OAuth\AuthProviderInterface;
use Psr\Http\Message\ResponseInterface;
use Spiral\Auth\AuthScope;
use Spiral\Auth\TokenStorageInterface;
use Spiral\Http\ResponseWrapper;
use Spiral\Router\Annotation\Route;
use Spiral\Session\SessionScope;

final readonly class LoginAction
{
    public function __construct(
        private ResponseWrapper $response,
    ) {}

    #[Route(route: 'auth/sso/login', methods: ['GET'], group: 'guest')]
    public function __invoke(
        AuthProviderInterface $auth,
        AuthScope $authScope,
        TokenStorageInterface $tokens,
        SuccessRedirect $successRedirect,
        SessionScope $session,
    ): ResponseInterface {
        if (!$auth->isAuthenticated()) {
            $url = $auth->getLoginUrl();

            return $this->response->redirect($url);
        }

        $authScope->start(
            $token = $tokens->create($auth->getUser()->jsonSerialize()),
        );

        return $successRedirect->makeResponse($token->getID());
    }
}
