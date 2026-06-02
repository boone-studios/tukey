<?php

namespace App\Domain\Traits;

trait HasEvents
{
    private array $events = [];

    public function recordEvent(string $eventClass, array $payload = []): void
    {
        $this->events[] = ['class' => $eventClass, 'payload' => $payload];
    }

    public function pullEvents(): array
    {
        $events = $this->events;
        $this->events = [];
        return $events;
    }

    public function hasEvents(): bool
    {
        return count($this->events) > 0;
    }
}
