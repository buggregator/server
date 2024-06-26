<?php

declare(strict_types=1);

namespace App\Integration\Auth0;

use Auth0\SDK\Contract\StoreInterface;
use Spiral\Session\SessionSectionInterface;

final readonly class SessionStore implements StoreInterface
{
    public function __construct(
        private SessionSectionInterface $session,
    ) {}

    /**
     * This has no effect when using sessions as the storage medium.
     *
     * @param bool $deferring whether to defer persisting the storage state
     *
     * @codeCoverageIgnore
     */
    public function defer(
        bool $deferring,
    ): void {}

    /**
     * Removes a persisted value identified by $key.
     *
     * @param string $key session key to delete
     */
    public function delete(
        string $key,
    ): void {
        $this->session->delete($key);
    }

    /**
     * Gets persisted values identified by $key.
     * If the value is not set, returns $default.
     *
     * @param string $key session key to set
     * @param mixed $default default to return if nothing was found
     *
     * @return mixed
     */
    public function get(
        string $key,
        $default = null,
    ) {
        return $this->session->get($key, $default);
    }

    /**
     * Removes all persisted values.
     */
    public function purge(): void
    {
        $this->session->clear();
    }

    /**
     * Persists $value on $_SESSION, identified by $key.
     *
     * @param string $key session key to set
     * @param mixed $value value to use
     */
    public function set(
        string $key,
        $value,
    ): void {
        $this->session->set($key, $value);
    }
}
