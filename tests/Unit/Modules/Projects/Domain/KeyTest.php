<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Projects\Domain;

use Modules\Projects\Domain\ValueObject\Key;
use PHPUnit\Framework\Attributes\DataProvider;
use Tests\TestCase;

final class KeyTest extends TestCase
{
    public static function validKeys(): iterable
    {
        yield ['project-key'];
        yield ['project_key'];
        yield ['projectkey'];
        yield ['projectkey123'];
        yield ['project-key-123'];
        yield ['project_key_123'];
        yield [\str_repeat('a', Key::MIN_LENGTH)];
        yield [\str_repeat('a', Key::MAX_LENGTH)];
    }

    #[DataProvider('validKeys')]
    public function testValid(string $key): void
    {
        $this->assertSame(Key::create($key)->value, $key);
    }
}
