<?php

namespace App\Domain;

use App\Domain\ValueObjects\Money;

class OrderItem extends Entity
{
    private string $productId;
    private string $productName;
    private Money $unitPrice;
    private int $quantity;

    public function __construct(string $productId, string $productName, Money $unitPrice, int $quantity)
    {
        $this->productId = $productId;
        $this->productName = $productName;
        $this->unitPrice = $unitPrice;
        $this->quantity = $quantity;
    }

    public static function create(string $productId, string $name, int $priceCents, int $quantity): self
    {
        $price = Money::fromCents($priceCents);
        return new self($productId, $name, $price, $quantity);
    }

    public function getId(): ?int
    {
        return $this->id;
    }

    public function validate(): bool
    {
        return $this->quantity > 0 && $this->unitPrice->getAmount() >= 0;
    }

    public function getSubtotal(): Money
    {
        return $this->unitPrice->multiply($this->quantity);
    }

    public function getProductId(): string
    {
        return $this->productId;
    }

    public function getQuantity(): int
    {
        return $this->quantity;
    }

    public function increaseQuantity(int $by): void
    {
        $this->quantity += $by;
    }
}
