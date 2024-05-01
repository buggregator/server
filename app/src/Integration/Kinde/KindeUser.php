<?php

declare(strict_types=1);

namespace App\Integration\Kinde;

use App\Application\OAuth\User;

final readonly class KindeUser extends User
{
    public function __construct(array $user)
    {
        parent::__construct(
            provider: 'kinde',
            username: $user['given_name'],
            avatar: $user['picture'],
            email: $user['email'],
        );
    }
}
