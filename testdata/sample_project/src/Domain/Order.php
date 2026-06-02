<?php

namespace App\Domain;

use App\Domain\Traits\Timestampable;
use App\Domain\Traits\HasEvents;
use App\Domain\Enums\OrderStatus;
use App\Domain\ValueObjects\Money;
use App\Domain\Contracts\Eventable;

class Order extends Entity implements Eventable
{
    use Timestampable;
    use HasEvents;

    private int $userId;
    private OrderStatus $status;
    private array $items = [];
    private Money $total;
    private string $currency;

    public function __construct(int $userId, string $currency)
    {
        $this->userId = $userId;
        $this->status = OrderStatus::Pending;
        $this->total = Money::zero($currency);
        $this->currency = $currency;
    }

    public function getId(): ?int
    {
        return $this->id;
    }

    public function validate(): bool
    {
        return $this->userId > 0 && count($this->items) > 0;
    }

    public function addItem(OrderItem $item): void
    {
        $this->items[] = $item;
        $this->total = $this->total->add($item->getSubtotal());
        $this->touch();
    }

    public function transition(OrderStatus $newStatus): void
    {
        if (!$this->status->canTransitionTo($newStatus)) {
            throw new \DomainException("Invalid status transition");
        }
        $this->status = $newStatus;
        $this->recordEvent('OrderStatusChanged', ['orderId' => $this->id, 'status' => $newStatus->value]);
        $this->touch();
    }

    public function cancel(): void
    {
        $this->transition(OrderStatus::Cancelled);
    }

    public function getStatus(): OrderStatus
    {
        return $this->status;
    }

    public function getTotal(): Money
    {
        return $this->total;
    }

    public function getUserId(): int
    {
        return $this->userId;
    }

    public function getItems(): array
    {
        return $this->items;
    }

    public function getEventName(): string
    {
        return $this->status->getEventName();
    }

    public function getPayload(): array
    {
        return ['orderId' => $this->id, 'userId' => $this->userId, 'status' => $this->status->value];
    }
}
