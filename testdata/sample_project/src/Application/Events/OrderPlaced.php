<?php

namespace App\Application\Events;

use App\Domain\Order;
use App\Domain\User;

class OrderPlaced
{
    private Order $order;
    private User $customer;
    private \DateTimeImmutable $occurredAt;

    public function __construct(Order $order, User $customer)
    {
        $this->order = $order;
        $this->customer = $customer;
        $this->occurredAt = new \DateTimeImmutable();
    }

    public function getOrder(): Order
    {
        return $this->order;
    }

    public function getCustomer(): User
    {
        return $this->customer;
    }

    public function getOccurredAt(): \DateTimeImmutable
    {
        return $this->occurredAt;
    }

    public function serialize(): array
    {
        return [
            'orderId'    => $this->order->getId(),
            'userId'     => $this->customer->getId(),
            'occurredAt' => $this->occurredAt->format(\DateTime::ATOM),
        ];
    }
}
