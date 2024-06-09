<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use App\Application\Domain\ValueObjects\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Entity;
use Cycle\Annotated\Annotation\Table\Index;
use Modules\Webhooks\Domain\ValueObject\Url;

#[Entity(
    repository: WebhookRepositoryInterface::class,
    table: 'webhooks',
)]
#[Index(columns: ['key'], unique: true)]
class Webhook
{
    /**  @internal */
    public function __construct(
        #[Column(type: 'string(36)', primary: true, typecast: 'uuid')]
        public Uuid $uuid,
        #[Column(type: 'string(50)')]
        public string $key,
        #[Column(type: 'string(50)')]
        public string $event,
        #[Column(type: 'text', typecast: Url::class)]
        public Url $url,
        #[Column(type: 'json', default: [], typecast: Json::class)]
        public Json $headers = new Json(),
        #[Column(type: 'boolean', default: false, typecast: 'bool')]
        public bool $verifySsl = false,
        #[Column(type: 'boolean', default: true, typecast: 'bool')]
        public bool $retryOnFailure = true,
    ) {}

    public function getUuid(): Uuid
    {
        return $this->uuid;
    }

    public function getHeaders(): Json
    {
        return $this->headers;
    }

    public function hasHeader(string $name): bool
    {
        return isset($this->headers[$name]);
    }

    public function getHeader(string $name): array
    {
        return $this->headers[$name] ?? [];
    }

    public function getHeaderLine(string $name): string
    {
        return \implode(',', $this->headers[$name] ?? []);
    }

    public function withHeader(string $name, string|\Stringable|array $value): self
    {
        $clone = clone $this;
        $clone->headers[$name] = \array_map(\strval(...), (array) $value);

        return $clone;
    }
}
