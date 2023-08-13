<?php

declare(strict_types=1);

namespace Modules\Attachments\Interfaces\Http\Controllers;

use App\Application\HTTP\Response\JsonResource;
use Modules\Attachments\Domain\Attachment;
use Psr\Http\Message\ServerRequestInterface;

/**
 * @property-read Attachment $data
 */
final class AttachmentResource extends JsonResource
{
    protected function mapData(ServerRequestInterface $request): array|\JsonSerializable
    {
        return [
            'uuid' => (string)$this->data->getUuid(),
            'size' => $this->data->getSize(),
            'mime' => $this->data->getMimeType(),
            'name' => $this->data->getFilename(),
            'download_uri' => \sprintf('/attachment/%s/download', $this->data->getUuid()),
        ];
    }
}
