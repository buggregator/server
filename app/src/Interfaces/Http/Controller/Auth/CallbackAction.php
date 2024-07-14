<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Controller\Auth;

use App\Application\OAuth\User;
use App\Application\Auth\SuccessRedirect;
use App\Application\OAuth\AuthProviderInterface;
use Psr\Http\Message\ResponseInterface;
use Spiral\Auth\AuthScope;
use Spiral\Auth\TokenStorageInterface;
use Spiral\Http\Exception\ClientException\UnauthorizedException;
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

        $user = $auth->getUser();
        if (!$user instanceof User) {
            throw new UnauthorizedException('User not found');
        }

        $authScope->start(
            $token = $tokens->create($user->jsonSerialize()),
        );

        return $successRedirect->makeResponse($token->getID());
    }
}
