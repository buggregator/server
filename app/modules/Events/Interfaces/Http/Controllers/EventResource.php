<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\HTTP\Response\JsonResource;
use Modules\Events\Domain\Event;
use Psr\Http\Message\ServerRequestInterface;

/**
 * @property-read Event $data
 */
final class EventResource extends JsonResource
{
    public function __construct(Event $data)
    {
        parent::__construct($data);
    }

    protected function mapData(ServerRequestInterface $request): array|\JsonSerializable
    {
        return [
            'uuid' => (string)$this->data->getUuid(),
            'type' => $this->data->getType(),
            'payload' => $this->data->getPayload(),
            'timestamp' => $this->data->getDate()->getTimestamp(),
            'project_id' => $this->data->getProjectId(),
        ];
    }
}
