<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Events;

use Modules\Events\Interfaces\Http\Resources\EventTypeCountResource;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class TypeCountsActionTest extends ControllerTestCase
{
    public function testTypeCountsByProject(): void
    {
        $this->createEvent(type: 'foo', project: 'alpha');
        $this->createEvent(type: 'foo', project: 'alpha');
        $this->createEvent(type: 'bar', project: 'alpha');
        $this->createEvent(type: 'baz', project: 'beta');

        $this->http
            ->typeCounts(project: 'alpha')
            ->assertOk()
            ->assertCollectionContainResources([
                new EventTypeCountResource(['type' => 'foo', 'cnt' => 2]),
                new EventTypeCountResource(['type' => 'bar', 'cnt' => 1]),
            ])
            ->assertCollectionMissingResources([
                new EventTypeCountResource(['type' => 'baz', 'cnt' => 1]),
            ]);
    }

    public function testTypeCountsByProjectAndType(): void
    {
        $this->createEvent(type: 'foo', project: 'alpha');
        $this->createEvent(type: 'foo', project: 'alpha');
        $this->createEvent(type: 'bar', project: 'alpha');

        $this->http
            ->typeCounts(type: 'foo', project: 'alpha')
            ->assertOk()
            ->assertCollectionContainResources([
                new EventTypeCountResource(['type' => 'foo', 'cnt' => 2]),
            ])
            ->assertCollectionMissingResources([
                new EventTypeCountResource(['type' => 'bar', 'cnt' => 1]),
            ]);
    }
}
