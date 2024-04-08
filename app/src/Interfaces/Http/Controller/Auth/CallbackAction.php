<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Controller\Auth;

use App\Application\Auth\SuccessRedirect;
use Auth0\SDK\Auth0;
use Psr\Http\Message\ResponseInterface;
use Spiral\Auth\AuthScope;
use Spiral\Auth\TokenStorageInterface;
use Spiral\Http\Request\InputManager;
use Spiral\Router\Annotation\Route;

final class CallbackAction
{
    #[Route(route: 'auth/sso/callback', methods: ['GET'], group: 'guest')]
    public function __invoke(
        Auth0 $auth,
        InputManager $input,
        AuthScope $authScope,
        TokenStorageInterface $tokens,
        SuccessRedirect $successRedirect,
    ): ResponseInterface {
        $auth->exchange(
            code: $input->query('code'),
            state: $input->query('state'),
        );

        $authScope->start(
            $token = $tokens->create($auth->getUser()),
        );

        return $successRedirect->makeResponse($token->getID());
    }
}
