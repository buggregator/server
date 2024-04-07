<?php

declare(strict_types=1);

namespace App\Application\HTTP\Response;

use App\Application\OAuth\User;

final class UserResource extends JsonResource
{
    public function __construct(
        private readonly User $user,
    ) {
        parent::__construct();
    }

    protected function mapData(): array
    {
        return [
            'username' => $this->user->getUsername(),
            'avatar' => $this->user->getAvatar(),
            'email' => $this->user->getEmail(),
        ];
    }
}
