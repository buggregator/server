<?php

declare(strict_types=1);

namespace App\Application\HTTP\Response;

use App\Application\OAuth\User;
use Psr\Http\Message\UriInterface;

final class UserResource extends JsonResource
{
    public function __construct(
        private readonly User $user,
        private readonly ?UriInterface $logoutUrl = null,
    ) {
        parent::__construct();
    }

    protected function mapData(): array
    {
        return [
            'provider' => $this->user->provider,
            'username' => $this->user->username,
            'avatar' => $this->user->avatar,
            'email' => $this->user->email,
            'logout' => $this->logoutUrl ? (string) $this->logoutUrl : null,
        ];
    }
}
