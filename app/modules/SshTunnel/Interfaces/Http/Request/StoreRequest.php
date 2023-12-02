<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Interfaces\Http\Request;

use Spiral\Filters\Attribute\CastingErrorMessage;
use Spiral\Filters\Attribute\Input\Post;
use Spiral\Filters\Attribute\Setter;
use Spiral\Filters\Model\Filter;
use Spiral\Filters\Model\FilterDefinitionInterface;
use Spiral\Filters\Model\HasFilterDefinition;
use Spiral\Validator\FilterDefinition;

final class StoreRequest extends Filter implements HasFilterDefinition
{
    #[Post]
    #[CastingErrorMessage('Name must be a string')]
    public string $name;

    #[Post]
    #[CastingErrorMessage('Host must be a string')]
    public string $host;

    #[Post]
    #[CastingErrorMessage('User must be a string')]
    public string $user = 'root';

    #[Post]
    #[Setter(filter: 'intval')]
    public int $port = 22;

    #[Post(key: 'private_key')]
    #[CastingErrorMessage('Password must be a string')]
    public string $privateKey;

    public function filterDefinition(): FilterDefinitionInterface
    {
        return new FilterDefinition([
            'name' => ['required', 'string'],
            'host' => ['required', 'string'],
            'user' => ['required', 'string'],
            'port' => ['int'],
            'privateKey' => ['required', 'string'],
        ]);
    }
}
