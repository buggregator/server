<?php

declare(strict_types=1);

namespace Application\Domain\ValueObjects;

use App\Application\Domain\ValueObjects\Uuid;
use PHPUnit\Framework\TestCase;

final class UuidTest extends TestCase
{
    public function testGenerate(): void
    {
        $uuid1 = Uuid::generate();
        $uuid2 = Uuid::generate();

        $this->assertNotEquals((string) $uuid1, (string) $uuid2);
    }

    public function testEquals(): void
    {
        $uuid1 = Uuid::generate();
        $uuid2 = Uuid::generate();

        $this->assertTrue($uuid1->equals($uuid1));
        $this->assertFalse($uuid1->equals($uuid2));
    }

    public function testFromString(): void
    {
        $uuid = Uuid::fromString($string = 'f47ac10b-58cc-4372-a567-0e02b2c3d479');

        $this->assertEquals($string, (string) $uuid);
    }
}
