<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Controller\Auth;

use App\Application\Auth\SuccessRedirect;
use App\Application\OAuth\AuthProviderInterface;
use Psr\Http\Message\ResponseInterface;
use Spiral\Auth\AuthScope;
use Spiral\Auth\TokenStorageInterface;
use Spiral\Http\Request\InputManager;
use Spiral\Router\Annotation\Route;

final readonly class CallbackAction
{
    #[Route(route: 'auth/sso/callback', methods: ['GET'], group: 'guest')]
    public function __invoke(
        AuthProviderInterface $auth,
        InputManager $input,
        AuthScope $authScope,
        TokenStorageInterface $tokens,
        SuccessRedirect $successRedirect,
    ): ResponseInterface {
        $auth->authenticate($input);

        $authScope->start(
            $token = $tokens->create($auth->getUser()->jsonSerialize()),
        );

        return $successRedirect->makeResponse($token->getID());
    }
}
