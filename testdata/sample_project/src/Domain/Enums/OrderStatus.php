<?php

namespace App\Domain\Enums;

use App\Domain\Contracts\Eventable;

enum OrderStatus: string implements Eventable
{
    case Pending = 'pending';
    case Processing = 'processing';
    case Shipped = 'shipped';
    case Delivered = 'delivered';
    case Cancelled = 'cancelled';
    case Refunded = 'refunded';

    public function getEventName(): string
    {
        return 'order.status.' . $this->value;
    }

    public function getPayload(): array
    {
        return ['status' => $this->value, 'timestamp' => time()];
    }

    public function isFinal(): bool
    {
        return in_array($this, [self::Delivered, self::Cancelled, self::Refunded]);
    }

    public function canTransitionTo(OrderStatus $next): bool
    {
        $allowed = [
            self::Pending->value    => [self::Processing->value, self::Cancelled->value],
            self::Processing->value => [self::Shipped->value, self::Cancelled->value],
            self::Shipped->value    => [self::Delivered->value],
            self::Delivered->value  => [self::Refunded->value],
        ];
        return in_array($next->value, $allowed[$this->value] ?? []);
    }
}
