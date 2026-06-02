<?php

namespace App\Application\Commands;

class PlaceOrderCommand
{
    private int $userId;
    private string $currency;
    private array $items;

    public function __construct(int $userId, string $currency, array $items)
    {
        $this->userId = $userId;
        $this->currency = $currency;
        $this->items = $items;
    }

    public function getUserId(): int
    {
        return $this->userId;
    }

    public function getCurrency(): string
    {
        return $this->currency;
    }

    public function getItems(): array
    {
        return $this->items;
    }

    public function validate(): bool
    {
        return $this->userId > 0 && count($this->items) > 0;
    }

    public function getItemCount(): int
    {
        return count($this->items);
    }
}
