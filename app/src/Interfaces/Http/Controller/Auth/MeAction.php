<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Controller\Auth;

use App\Application\HTTP\Response\ResourceInterface;
use App\Application\HTTP\Response\UserResource;
use App\Application\OAuth\ActorProvider;
use App\Application\OAuth\AuthProviderInterface;
use App\Application\OAuth\User;
use Spiral\Auth\AuthScope;
use Spiral\Router\Annotation\Route;

final readonly class MeAction
{
    #[Route(route: 'me', methods: ['GET'], group: 'api')]
    public function __invoke(
        AuthScope $authScope,
        AuthProviderInterface $auth,
    ): ResourceInterface {
        /** @var User $actor */
        $actor = $authScope->getActor() ?? ActorProvider::getGuestPayload();

        return new UserResource($actor, $auth->getLogoutUrl());
    }
}
