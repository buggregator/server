<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Interfaces\Http\Resources;

use App\Application\HTTP\Response\JsonResource;
use Modules\SshTunnel\Domain\Connection;

/**
 * @property-read Connection $data
 */
final class SshConnectionResource extends JsonResource
{
    public function __construct(Connection $data)
    {
        parent::__construct($data);
    }

    protected function mapData(): array|\JsonSerializable
    {
        return [
            'uuid' => (string)$this->data->uuid,
            'name' => $this->data->name,
            'host' => $this->data->host,
            'port' => $this->data->port,
            'user' => $this->data->user,
            'private_key' => $this->data->privateKey,
        ];
    }
}
