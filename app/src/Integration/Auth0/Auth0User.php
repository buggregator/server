<?php

declare(strict_types=1);

namespace App\Integration\Auth0;

use App\Application\OAuth\User;

final readonly class Auth0User extends User
{
    public function __construct(array $user)
    {
        parent::__construct(
            provider: 'auth0',
            username: $user['nickname'],
            avatar: $user['picture'],
            email: $user['email'],
        );
    }
}
